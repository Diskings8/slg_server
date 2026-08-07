package session_gateways

// switchForward — 客户端请求转发：按 MsgID 分类目标节点，调后端 gRPC，回包客户端。
//
// 包协议约定（客户端 ↔ gateway）：
//   - 上行：TCP header(msgID) + body = 序列化的请求 proto（如 LoginAccountReq）
//   - 下行：TCP header(msgID) + body = 序列化的 common.MessagePacket（err_code + body）
//
// login 协议段 → login 节点 Do；game 协议段 → 进服后经双向流到 game（见 session.forward.game.func.go）。

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_account"
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
	case isGameMsgID(msgID):
		s.forwardToGame(packet)
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

// isGameMsgID 是否 game 协议段（业务 + 相机）
func isGameMsgID(msgID pb_protocol.MsgID) bool {
	switch id := int32(msgID); {
	case id >= 1000001 && id <= 1009999: // Game 业务协议段
		return true
	case id >= 42200 && id <= 42299: // Camera 相机段
		return true
	}
	return false
}

// forwardToLogin 转发 login 协议到 login 节点 Do（同步请求-响应）
//
// 进入区服（LoginEnterServer）：转发前捕获 server_id，成功后捕获 role_id，供后续 game 协议路由。
func (s *Session) forwardToLogin(packet *packets.Packet) {
	cli := gateway_rpc_clients.Client().Login()
	if cli == nil {
		loggers.Logger.Warn("login node not connected", zap.Uint32("msg_id", packet.MsgID))
		s.writeErrorPacket(packet.Seq, pb_protocol.MsgID(packet.MsgID), pb_error_code.ErrorCode_SystemBusy, "login node not connected")
		return
	}

	msgID := pb_protocol.MsgID(packet.MsgID)
	if msgID == pb_protocol.MsgID_LoginEnterServer {
		s.captureEnterServerID(packet)
	}

	// 从会话持有的全局 ctx 派生 + 超时：关停时在途 login 请求一并取消
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	resp, err := cli.Do(ctx, &pb_common.NodePacket{
		MsgId: msgID,
		Message: &pb_common.MessagePacket{
			Body:           packet.Body,
			GatewayNodeId:  nodeID, // 携带本 gateway 标识，login 记录角色所在节点并广播
		},
	})
	if err != nil {
		loggers.Logger.Warn("forward to login failed",
			zap.Uint32("msg_id", packet.MsgID), zap.Error(err))
		s.writeErrorPacket(packet.Seq, msgID, pb_error_code.ErrorCode_SystemBusy, err.Error())
		return
	}

	if msgID == pb_protocol.MsgID_LoginEnterServer {
		s.captureEnterServerRoleID(resp)
	}
	s.writeNodePacket(packet.Seq, resp)
}

// captureEnterServerID 从 EnterServer 请求体捕获 server_id（转发前）
func (s *Session) captureEnterServerID(packet *packets.Packet) {
	var req pb_account.EnterServerReq
	if err := proto.Unmarshal(packet.Body, &req); err == nil && req.GetServerId() > 0 {
		s.serverID = req.GetServerId()
	}
}

// captureEnterServerRoleID 从 EnterServer 响应体捕获 role_id（成功后）
func (s *Session) captureEnterServerRoleID(resp *pb_common.NodePacket) {
	var out pb_account.EnterServerResp
	if err := proto.Unmarshal(resp.GetMessage().GetBody(), &out); err == nil && out.GetRoleId() > 0 {
		s.roleID = out.GetRoleId()
		// 登记到下推注册表：其他节点可经 GatewayService.Stream 按 roleID 定位本连接并下推
		defaultSessionManager.Register(s.roleID, s)
	}
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
	if err := s.writePacket(seq, &packets.Packet{
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
	if err := s.writePacket(seq, &packets.Packet{
		MsgID: uint32(msgID),
		Body:  body,
	}); err != nil {
		loggers.Logger.Warn("write error to client failed", zap.Error(err))
	}
}
