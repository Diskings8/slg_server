package game_rpc_clients

import (
	"context"
	"time"

	"server.slg.com/services/game/game_internals/game_rpc_clients/worldmap_client"
)

// Init 初始化所有 RPC 客户端，并启动统一的后台连接维护协程
// 懒连接：此处只创建客户端对象，不主动拨号
func (m *gameRpcClientMap) Init(parentCtx context.Context) {
	m.ctx = parentCtx
	m.worldmap = worldmap_client.New(parentCtx)
	go m.watchConnects()
}

// watchConnects 用单一 timer ticker 统一维护所有客户端的连接
func (m *gameRpcClientMap) watchConnects() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.closeAll()
			return
		case <-ticker.C:
			m.maintainAll()
		}
	}
}

// maintainAll 只维护已激活（曾连接过）的客户端
// 未使用过的懒连接客户端由业务访问时触发建立
func (m *gameRpcClientMap) maintainAll() {
	if m.worldmap != nil && m.worldmap.Active() {
		m.worldmap.Connect()
	}
}

// closeAll 遍历所有客户端关闭连接
func (m *gameRpcClientMap) closeAll() {
	if m.worldmap != nil {
		m.worldmap.Close()
	}
}

// WorldMap 获取 worldmap 客户端，首次访问时触发连接（懒连接，幂等）
func (m *gameRpcClientMap) WorldMap() *worldmap_client.Client {
	if m.worldmap != nil {
		m.worldmap.Connect()
	}
	return m.worldmap
}
