package login_game_clients

import (
	"context"
	"strconv"
	"sync"

	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
)

// RoleCreator 建角能力接口，供 handler 依赖注入，测试可 mock
type RoleCreator interface {
	CreateRole(ctx context.Context, serverID uint32, req *pb_common.CreateRoleReq) (*pb_common.CreateRoleRsp, error)
}

var _ RoleCreator = (*GameClient)(nil)

// GameClient 按 server_id 访问对应 game 节点的门面
//
// 一个 game 进程 = 一个区服（instance == server_id），这里按 server_id 懒创建独立
// ClientHandler（instance 感知的 etcd 发现），镜像 game 连 worldmap 的 WorldMapByInstance 模式。
type GameClient struct {
	mu      sync.Mutex
	clients map[uint32]*rpc_handlers.ClientHandler // server_id → hub
}

func NewGameClient() *GameClient {
	return &GameClient{clients: make(map[uint32]*rpc_handlers.ClientHandler)}
}

// hub 按 server_id 取（懒创建）ClientHandler
func (c *GameClient) hub(serverID uint32) *rpc_handlers.ClientHandler {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h, ok := c.clients[serverID]; ok {
		return h
	}
	h := rpc_handlers.NewClientHandler(strconv.FormatUint(uint64(serverID), 10))
	c.clients[serverID] = h
	return h
}

// Close 关闭所有底层连接（进程退出时调用）
func (c *GameClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, h := range c.clients {
		h.Close()
	}
	c.clients = make(map[uint32]*rpc_handlers.ClientHandler)
}
