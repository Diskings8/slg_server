package game_streams

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_internals/gate_stream"
	"server.slg.com/services/game/game_logics"
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

	// 记录登录统计（跨日登录天数/最后登录时间）+ 取主城坐标（供 worldmap 视野流握手）
	mainCityMapID := int32(0)
	if poller, role, err := game_roles.GetRole(roleID); err == nil {
		role.Login()
		// 登录惰性结算：建造/升级到期的建筑置为 Completed（城市完成→落校场+同步队列）；
		// 资源地产出也在此补结算（跨会话离线产出入账）
		settled := game_logics.SettleBuildings(role, roleID)
		if game_logics.SettleRoleResources(role, roleID) {
			settled = true
		}
		if settled {
			poller.Save()
		}
		if main := role.GetBuildings().GetMainCity(); main != nil {
			mainCityMapID = main.MapID
		}
		poller.Save()
		poller.Release()
	}

	// 玩家建立到 worldmap 的视野流（cores push 经此流下推回客户端）
	// mapID: 玩家所在地图位置（主城核心格，无主城时为 0）
	if err := game_rpc_clients.WorldMap().ConnectRoleStream(stream.Context(), roleID, mainCityMapID); err != nil {
		loggers.Logger.Warn("connect worldmap stream failed",
			zap.Uint64("role_id", roleID), zap.Error(err))
	}

	// 阻塞等待连接断开（Recv 在后台 goroutine 中持续处理消息）
	gConn.WaitDone()

	// ── 连接断开，清理 ──
	gate_stream.GateClose(roleID)
	game_rpc_clients.WorldMap().CloseRoleStream(roleID)

	poller, roleTmp, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	roleTmp.Offline()
	poller.Save()
	poller.Release()
	
	return nil
}
