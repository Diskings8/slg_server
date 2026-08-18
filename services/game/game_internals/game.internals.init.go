package game_internals

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_internals/gate_stream"
	"server.slg.com/services/game/game_internals/stream_consumers"
	"server.slg.com/services/game/game_logics"

	"go.uber.org/zap"
)

func Init(ctx context.Context) {
	gate_stream.Init(ctx)
	stream_consumers.Init(ctx)
	game_rpc_clients.Init(ctx)
	wireRoleUnionIndexSync()
}

// wireRoleUnionIndexSync 联盟成员变更 → worldmap RoleUnionIndex 同步（best-effort，worldmap 未连时跳过）
func wireRoleUnionIndexSync() {
	game_logics.SetRoleUnionIndexSyncFunc(func(roleID, unionID uint64) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := game_rpc_clients.WorldMap().SyncRoleUnion(ctx, &pb_worldmap.SyncRoleUnionReq{
			RoleId: roleID, UnionId: unionID,
		}); err != nil {
			loggers.Logger.Warn("sync role union to worldmap failed",
				zap.Uint64("role_id", roleID), zap.Error(err))
		}
	})
}

func ShutDown() {
	gate_stream.ShutDown()
	game_rpc_clients.ShutDown()
}
