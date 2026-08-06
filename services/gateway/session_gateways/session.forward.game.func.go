package session_gateways

// game 协议段转发：进服后维护一条到 game 的双向流。
//
// 链路：
//   客户端 game 请求 → forwardToGame → gameStream.Send(GameServiceNodePacketReq{RoleId, Packet})
//   game 下推 → gameStream.Recv → writeNodePacket → 客户端 TCP（MessagePacket 信封）
//   TCP 断开 / 流断开 → cleanupGameStream → cancel → game 侧 gateConnectDo 收 Done → Offline 下线

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/netconn/packets"
	"server.slg.com/common/loggers"
	"server.slg.com/services/gateway/gateway_internals/gateway_rpc_clients"
)

// forwardToGame 上行转发：把客户端 game 协议经双向流发给 game（首次请求时懒连接）
func (s *Session) forwardToGame(packet *packets.Packet) {
	msgID := pb_protocol.MsgID(packet.MsgID)

	// 未进服（无 serverID/roleID）→ 拒绝
	if s.serverID == 0 || s.roleID == 0 {
		s.writeErrorPacket(packet.Seq, msgID, pb_error_code.ErrorCode_Failed, "not entered game")
		return
	}

	// 懒连接 game 流
	s.streamMu.Lock()
	stream := s.gameStream
	s.streamMu.Unlock()
	if stream == nil {
		if err := s.connectGameStream(s.serverID, s.roleID); err != nil {
			loggers.Logger.Warn("connect game stream failed",
				zap.Uint32("server_id", s.serverID), zap.Error(err))
			s.writeErrorPacket(packet.Seq, msgID, pb_error_code.ErrorCode_SystemBusy, err.Error())
			return
		}
		s.streamMu.Lock()
		stream = s.gameStream
		s.streamMu.Unlock()
		if stream == nil {
			return
		}
	}

	s.streamMu.Lock()
	s.lastSeq = packet.Seq
	s.streamMu.Unlock()

	if err := stream.Send(&pb_game.GameServiceNodePacketReq{
		RoleId: s.roleID,
		Packet: &pb_common.NodePacket{
			MsgId:   msgID,
			Message: &pb_common.MessagePacket{Body: packet.Body},
		},
	}); err != nil {
		loggers.Logger.Warn("forward to game failed", zap.Uint32("msg_id", packet.MsgID), zap.Error(err))
		s.cleanupGameStream()
		s.writeErrorPacket(packet.Seq, msgID, pb_error_code.ErrorCode_SystemBusy, err.Error())
	}
}

// connectGameStream 打开到 game[serverID] 的双向流并握手（首包携带 roleID）
func (s *Session) connectGameStream(serverID uint32, roleID uint64) error {
	cli := gateway_rpc_clients.Client().Game(serverID).GetGameServiceClient()
	if cli == nil {
		return fmt.Errorf("game[%d] not connected", serverID)
	}

	streamCtx, cancel := context.WithCancel(context.Background())
	stream, err := cli.Stream(streamCtx)
	if err != nil {
		cancel()
		return fmt.Errorf("game[%d] stream open failed: %w", serverID, err)
	}
	if err := stream.Send(&pb_game.GameServiceNodePacketReq{RoleId: roleID}); err != nil {
		cancel()
		_ = stream.CloseSend()
		return fmt.Errorf("game[%d] stream handshake failed: %w", serverID, err)
	}

	s.streamMu.Lock()
	s.gameStream = stream
	s.streamCancel = cancel
	s.streamMu.Unlock()

	go s.recvFromGame()
	return nil
}

// recvFromGame 下行接收：game 下推 → 客户端 TCP
func (s *Session) recvFromGame() {
	defer func() {
		if e := recover(); e != nil {
			loggers.Logger.Error(fmt.Sprintf("session recvFromGame error :%+v", e))
		}
	}()

	s.streamMu.Lock()
	stream := s.gameStream
	s.streamMu.Unlock()
	if stream == nil {
		return
	}

	for {
		rsp, err := stream.Recv()
		if err != nil {
			loggers.Logger.Info("game stream down", zap.Uint64("role_id", s.roleID), zap.Error(err))
			s.cleanupGameStream()
			return
		}
		if rsp == nil || rsp.GetPacket() == nil {
			continue
		}
		msgID := rsp.GetPacket().GetMsgId()
		seq := uint32(0)
		if !isPushMsgID(msgID) {
			// 请求-响应：回最近一次上行请求的 seq（客户端按 seq 匹配）
			s.streamMu.Lock()
			seq = s.lastSeq
			s.streamMu.Unlock()
		}
		s.writeNodePacket(seq, rsp.GetPacket())
	}
}

// cleanupGameStream 关闭 game 流（幂等）
func (s *Session) cleanupGameStream() {
	s.streamMu.Lock()
	stream := s.gameStream
	cancel := s.streamCancel
	s.gameStream = nil
	s.streamCancel = nil
	s.streamMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if stream != nil {
		_ = stream.CloseSend()
	}
}

// isPushMsgID 是否服务端主动下推（无请求 seq，回包 seq 用 0）
func isPushMsgID(msgID pb_protocol.MsgID) bool {
	return int32(msgID) >= 10000000 // 推送段：10000001（维护/版本）、11000001（地图/行军）
}
