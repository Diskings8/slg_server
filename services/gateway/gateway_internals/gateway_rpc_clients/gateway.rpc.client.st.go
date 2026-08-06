package gateway_rpc_clients

import (
	"sync"

	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
)

// defaultClient 包级单例
var defaultClient = &GatewayRpcClient{}

// GatewayRpcClient gateway 出向 RPC 客户端门面
//
// login 节点为跨服全局单例（instance=0），懒连接即可；后续 game 节点按 server_id 建独立 hub。
type GatewayRpcClient struct {
	mu          sync.Mutex
	loginHub    *rpc_handlers.ClientHandler // login 节点（instance 0）
	loginClient pb_account.AccountServiceClient
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
