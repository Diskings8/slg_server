package game_rpc_clients

import (
	"context"

	"server.slg.com/common/conns/rpcconn/rpc_conns"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/services/game/game_internals/game_rpc_clients/worldmap_client"
)

// Init 初始化所有 RPC 客户端门面
//
// 底层连接维护在 rpcconn：rpc_conns 连接池 + 生成的 hub（按 nodeType + instance 感知发现）。
// 懒连接：此处只创建门面对象，首次业务访问时由 hub 触发拨号。
func (m *gameRpcClientMap) Init(parentCtx context.Context) {
	m.ctx = parentCtx
	m.hub = rpc_handlers.NewClientHandler(*common_globals.CommonGlobalVarInstance)
	m.worldmap = worldmap_client.New(m.hub)
}

// WorldMap 获取 worldmap 客户端门面
func (m *gameRpcClientMap) WorldMap() *worldmap_client.Client {
	return m.worldmap
}

// ShutDown 关闭所有客户端连接
func (m *gameRpcClientMap) ShutDown() {
	if m.hub != nil {
		m.hub.Close()
	}
	// 连接池为进程级单例，进程关闭时统一释放池中所有连接
	rpc_conns.CloseAll()
}

// WorldMap 包级访问器：获取 worldmap 客户端门面
func WorldMap() *worldmap_client.Client {
	return GameRpcClientHandler.WorldMap()
}
