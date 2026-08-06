package gateway_rpc_clients

import (
	"strconv"
	"sync"

	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
)

// defaultClient 包级单例
var defaultClient = &GatewayRpcClient{}

// GatewayRpcClient gateway 出向 RPC 客户端门面
//
// login 节点为跨服全局单例（instance=0），懒连接即可；game 节点按 server_id 懒建独立 hub
// （instance = serverID，一个 game 进程 = 一个区服）。
type GatewayRpcClient struct {
	mu          sync.Mutex
	loginHub    *rpc_handlers.ClientHandler        // login 节点（instance 0）
	loginClient pb_account.AccountServiceClient
	gameHubs    map[uint32]*rpc_handlers.ClientHandler // serverID → game hub
}

// Client 获取单例
func Client() *GatewayRpcClient {
	return defaultClient
}

// Login 获取 login 节点 AccountService 客户端（懒连接，instance=0）
func (c *GatewayRpcClient) Login() pb_account.AccountServiceClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loginClient != nil {
		return c.loginClient
	}
	if c.loginHub == nil {
		c.loginHub = rpc_handlers.NewClientHandler("0")
	}
	c.loginClient = c.loginHub.GetAccountServiceClient()
	return c.loginClient
}

// SetLoginClient 覆盖 login 客户端（测试注入 / 节点切换）
func (c *GatewayRpcClient) SetLoginClient(cli pb_account.AccountServiceClient) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loginClient = cli
}

// Game 获取指定 serverID 的 game 节点 ClientHandler（懒创建，instance = serverID）
func (c *GatewayRpcClient) Game(serverID uint32) *rpc_handlers.ClientHandler {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h, ok := c.gameHubs[serverID]; ok {
		return h
	}
	if c.gameHubs == nil {
		c.gameHubs = make(map[uint32]*rpc_handlers.ClientHandler)
	}
	h := rpc_handlers.NewClientHandler(strconv.FormatUint(uint64(serverID), 10))
	c.gameHubs[serverID] = h
	return h
}
