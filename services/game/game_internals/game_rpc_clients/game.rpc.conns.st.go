package game_rpc_clients

import (
	"context"

	"server.slg.com/common/conns/rpcconn/rpc_handlers"
	"server.slg.com/services/game/game_internals/game_rpc_clients/worldmap_client"
)

var GameRpcClientHandler = &gameRpcClientMap{}

func Init(ctx context.Context) {
	GameRpcClientHandler.Init(ctx)
}

func ShutDown() {
	GameRpcClientHandler.ShutDown()
}

// gameRpcClientMap 聚合所有 RPC 客户端门面
// 底层连接维护委托给 rpcconn（连接池 + 生成 hub，instance 感知发现），此处只持有门面
type gameRpcClientMap struct {
	ctx      context.Context
	hub      *rpc_handlers.ClientHandler
	worldmap *worldmap_client.Client
}
