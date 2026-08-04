package game_rpc_clients

import (
	"context"
	"sync"

	"server.slg.com/services/game/game_internals/game_rpc_clients/battle_record_client"
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
// 底层连接维护委托给 rpcconn（连接池 + 生成 hub，instance 感知发现），此处只持有门面。
// 默认 instance（本服）作为 map 中的一个条目；显式指定其他 instance 时按 instance 懒创建独立 hub + 门面复用。
// 后续新增连接类型（game/gateway 等）时，各按实例建一个独立 map（如 gamesByInstance）。
type gameRpcClientMap struct {
	ctx                 context.Context
	mu                  sync.RWMutex
	instance            string                                  // 默认 instance（本服），RPC 发现用它对齐
	worldmapsByInstance map[string]*worldmap_client.Client      // instance → worldmap 门面（含默认，懒创建）
	battleRecordsByInst map[string]*battle_record_client.Client // instance → battle_record 门面（战报查询）
}
