package game_entitys

import (
	"context"

	"server.slg.com/common/conns/dbconn"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// Init 初始化实体数据层（子模块 AutoMigrate + PollerManager）
func Init(ctx context.Context) {
	game_roles.Init(ctx, dbconn.GetWriteDbConn())
}

// ShutDown 关闭实体数据层（关闭 PollerManager）
func ShutDown() {
	_ = game_roles.Close()
}
