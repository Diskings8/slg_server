package login_servers

import (
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_internals/login_game_clients"
	"server.slg.com/services/login/login_internals/login_servers"
	"server.slg.com/services/login/login_internals/login_tokens"
)

// LoginServerHandler 登录 RPC 服务（CreateAccount/LoginAccount/ServerList/EnterServer）
var LoginServerHandler = &LoginServer{}

// LoginServer 账号服务实现
type LoginServer struct {
	pb_account.UnimplementedAccountServiceServer
	accountStore *login_accounts.AccountStore
	channelStore *login_channels.ChannelStore
	serverStore  *login_servers.ServerStore
	tokens       *login_tokens.TokenManager
	gameClient   login_game_clients.RoleCreator
}

// SetStore 注入依赖（main.go AsyncInit 时调用）
func (s *LoginServer) SetStore(accountStore *login_accounts.AccountStore, channelStore *login_channels.ChannelStore, serverStore *login_servers.ServerStore, tokens *login_tokens.TokenManager, gameClient login_game_clients.RoleCreator) {
	s.accountStore = accountStore
	s.channelStore = channelStore
	s.serverStore = serverStore
	s.tokens = tokens
	s.gameClient = gameClient
}
