package game_streams

import (
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_internals/gate_stream"
)

// GameStreamHandler 供 main.go 注册 gRPC 服务
var GameStreamHandler = &GameStream{}

// GameStream GameService 流处理
type GameStream struct {
	pb_game.UnimplementedGameServiceServer
}

// Stream GameService 流式连接入口
func (gs *GameStream) Stream(stream pb_game.GameService_StreamServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	if req.GetRoleId() < 1 {
		return nil
	}
	return gs.gateConnectDo(req.GetRoleId(), stream)
}

// gateConnectDo 处理 Gateway 连接生命周期
func (gs *GameStream) gateConnectDo(roleID uint64, stream pb_game.GameService_StreamServer) error {
	// 注册连接到网关流
	gConn, err := gate_stream.GateJoin(roleID, false, stream, gs.Recv)
	if err != nil {
		return err
	}

	// 阻塞等待连接断开（Recv 在后台 goroutine 中持续处理消息）
	gConn.WaitDone()

	// ── 连接断开，清理 ──
	gate_stream.GateClose(roleID)

	poller, err := game_roles.GetPoller(roleID)
	if err != nil {
		return err
	}
	roleTmp, err := poller.Get()
	if err != nil {
		poller.Release()
		return err
	}
	roleTmp.Offline()
	poller.Release()
	poller.SaveSync()

	return nil
}
