package game_rpc_clients

import (
	"context"

	"server.slg.com/common/conns/rpcconn/rpc_conns"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/services/game/game_internals/game_rpc_clients/battle_record_client"
	"server.slg.com/services/game/game_internals/game_rpc_clients/worldmap_client"
)

// Init 初始化所有 RPC 客户端门面
//
// 底层连接维护在 rpcconn：rpc_conns 连接池 + 生成的 hub（按 nodeType + instance 感知发现）。
// 懒连接：此处只创建默认 instance 的门面对象，首次业务访问时由 hub 触发拨号。
func (m *gameRpcClientMap) Init(parentCtx context.Context) {
	m.ctx = parentCtx
	m.instance = *common_globals.CommonGlobalVarInstance
	m.worldmapsByInstance = make(map[string]*worldmap_client.Client)
	m.worldmapsByInstance[m.instance] = worldmap_client.New(rpc_handlers.NewClientHandler(m.instance))
	m.battleRecordsByInst = make(map[string]*battle_record_client.Client)
	m.battleRecordsByInst[m.instance] = battle_record_client.New(rpc_handlers.NewClientHandler(m.instance))
}

// WorldMap 获取 worldmap 客户端门面（默认连接本服 instance 的 worldmap）
func (m *gameRpcClientMap) WorldMap() *worldmap_client.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.worldmapsByInstance[m.instance]
}

// WorldMapByInstance 获取指定 instance 的 worldmap 客户端门面
//
// 默认 instance（本服）即 worldmapsByInstance 中的条目；其他 instance 首次访问时
// 按 instance 懒创建独立 hub + 门面，后续复用。适合跨服/指定节点通信。
func (m *gameRpcClientMap) WorldMapByInstance(instance string) *worldmap_client.Client {
	if instance == "" {
		instance = m.instance
	}

	m.mu.RLock()
	c, ok := m.worldmapsByInstance[instance]
	m.mu.RUnlock()
	if ok {
		return c
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.worldmapsByInstance[instance]; ok {
		return c
	}
	c = worldmap_client.New(rpc_handlers.NewClientHandler(instance))
	m.worldmapsByInstance[instance] = c
	return c
}

// ShutDown 关闭所有客户端连接
func (m *gameRpcClientMap) ShutDown() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, c := range m.worldmapsByInstance {
		c.Close()
	}
	for _, c := range m.battleRecordsByInst {
		c.Close()
	}
	// 连接池为进程级单例，进程关闭时统一释放池中所有连接
	rpc_conns.CloseAll()
}

// BattleRecord 获取 battle_record 客户端门面（默认连接本服 instance）
func (m *gameRpcClientMap) BattleRecord() *battle_record_client.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.battleRecordsByInst[m.instance]
}

// WorldMap 包级访问器：获取 worldmap 客户端门面（默认 instance）
func WorldMap() *worldmap_client.Client {
	return GameRpcClientHandler.WorldMap()
}

// WorldMapByInstance 包级访问器：获取指定 instance 的 worldmap 客户端门面
func WorldMapByInstance(instance string) *worldmap_client.Client {
	return GameRpcClientHandler.WorldMapByInstance(instance)
}

// BattleRecord 包级访问器：获取 battle_record 客户端门面（默认 instance）
func BattleRecord() *battle_record_client.Client {
	return GameRpcClientHandler.BattleRecord()
}
