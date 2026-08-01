package game_rpc_clients

import (
	"context"

	"server.slg.com/services/game/game_internals/game_rpc_clients/worldmap_client"
)

var GameRpcClientHandler = &gameRpcClientMap{}

func Init(ctx context.Context) {
	GameRpcClientHandler.Init(ctx)
}

// gameRpcClientMap 聚合所有 RPC 客户端，统一维护连接
type gameRpcClientMap struct {
	ctx      context.Context
	worldmap *worldmap_client.Client
}
