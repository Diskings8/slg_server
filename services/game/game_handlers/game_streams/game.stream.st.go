package game_streams

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
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

	// 记录登录统计（跨日登录天数/最后登录时间）
	if poller, role, err := game_role_handler.GetRole(roleID); err == nil {
		role.Login()
		poller.Save()
		poller.Release()
	}

	// 玩家建立到 worldmap 的视野流（cores push 经此流下推回客户端）
	// mapID: 玩家所在地图位置，TODO 主城落位后从角色数据获取
	if err := game_rpc_clients.WorldMap().ConnectRoleStream(stream.Context(), roleID, 0); err != nil {
		loggers.Logger.Warn("connect worldmap stream failed",
			zap.Uint64("role_id", roleID), zap.Error(err))
	}

	// 阻塞等待连接断开（Recv 在后台 goroutine 中持续处理消息）
	gConn.WaitDone()

	// ── 连接断开，清理 ──
	gate_stream.GateClose(roleID)
	game_rpc_clients.WorldMap().CloseRoleStream(roleID)

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
