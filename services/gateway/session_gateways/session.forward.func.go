package session_gateways

// switchForward — 客户端请求转发：按 MsgID 分类目标节点，调后端 gRPC，回包客户端。
//
// 包协议约定（客户端 ↔ gateway）：
//   - 上行：TCP header(msgID) + body = 序列化的请求 proto（如 LoginAccountReq）
//   - 下行：TCP header(msgID) + body = 序列化的 common.MessagePacket（err_code + body）
//
// 当前支持 login 协议段（转发到 login 节点的 Do）；game 协议段（推流下行）后续接入。

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/netconn/packets"
	"server.slg.com/common/loggers"
	"server.slg.com/services/gateway/gateway_internals/gateway_rpc_clients"
)

// switchForward 按 MsgID 路由并转发
func (s *Session) switchForward(packet *packets.Packet) {
	msgID := pb_protocol.MsgID(packet.MsgID)
	switch {
	case isLoginMsgID(msgID):
		s.forwardToLogin(packet)
	default:
		loggers.Logger.Warn("gateway unsupported protocol", zap.Uint32("msg_id", packet.MsgID))
		s.writeErrorPacket(packet.Seq, msgID, pb_error_code.ErrorCode_ProtocolNotFound, "protocol not supported")
	}
}

// isLoginMsgID 是否 login 协议段
func isLoginMsgID(msgID pb_protocol.MsgID) bool {
	switch msgID {
	case pb_protocol.MsgID_LoginCreateAccount,
		pb_protocol.MsgID_LoginAccount,
		pb_protocol.MsgID_LoginServerList,
		pb_protocol.MsgID_LoginEnterServer:
		return true
	}
	return false
}

// forwardToLogin 转发 login 协议到 login 节点 Do（同步请求-响应）
func (s *Session) forwardToLogin(packet *packets.Packet) {
	cli := gateway_rpc_clients.Client().Login()
	if cli == nil {
		loggers.Logger.Warn("login node not connected", zap.Uint32("msg_id", packet.MsgID))
		s.writeErrorPacket(packet.Seq, pb_protocol.MsgID(packet.MsgID), pb_error_code.ErrorCode_SystemBusy, "login node not connected")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := cli.Do(ctx, &pb_common.NodePacket{
		MsgId: pb_protocol.MsgID(packet.MsgID),
		Message: &pb_common.MessagePacket{
			Body: packet.Body,
		},
	})
	if err != nil {
		loggers.Logger.Warn("forward to login failed",
			zap.Uint32("msg_id", packet.MsgID), zap.Error(err))
		s.writeErrorPacket(packet.Seq, pb_protocol.MsgID(packet.MsgID), pb_error_code.ErrorCode_SystemBusy, err.Error())
		return
	}

	s.writeNodePacket(packet.Seq, resp)
}

// writeNodePacket 把后端 NodePacket 的 Message（即客户端信封 MessagePacket）回写客户端 TCP
func (s *Session) writeNodePacket(seq uint32, resp *pb_common.NodePacket) {
	msg := resp.GetMessage()
	if msg == nil {
		msg = &pb_common.MessagePacket{ErrCode: pb_error_code.ErrorCode_Failed}
	}
	body, err := proto.Marshal(msg)
	if err != nil {
		loggers.Logger.Warn("marshal response packet failed", zap.Error(err))
		return
	}
	if err := s.conn.WriteToConn(seq, &packets.Packet{
		MsgID: uint32(resp.GetMsgId()),
		Body:  body,
	}); err != nil {
		loggers.Logger.Warn("write response to client failed", zap.Error(err))
	}
}

// writeErrorPacket 网关侧错误（节点不可达等）直接构造错误信封回写
func (s *Session) writeErrorPacket(seq uint32, msgID pb_protocol.MsgID, code pb_error_code.ErrorCode, devMsg string) {
	body, err := proto.Marshal(&pb_common.MessagePacket{
		ErrCode: code,
		DevMsg:  devMsg,
	})
	if err != nil {
		return
	}
	if err := s.conn.WriteToConn(seq, &packets.Packet{
		MsgID: uint32(msgID),
		Body:  body,
	}); err != nil {
		loggers.Logger.Warn("write error to client failed", zap.Error(err))
	}
}
