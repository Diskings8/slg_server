package login_internals

import (
	"go.uber.org/zap"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/login/login_handlers/login_servers"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_internals/login_game_clients"
	login_servers_store "server.slg.com/services/login/login_internals/login_servers"
	"server.slg.com/services/login/login_internals/login_tokens"
)

var gameClient *login_game_clients.GameClient

func Init() {
	accountStore := login_accounts.NewAccountStore(dbconn.GetWriteDbConn())
	if err := accountStore.Migrate(); err != nil {
		loggers.Logger.Fatal("账号表结构初始化失败", zap.Error(err))
	}

	channelStore := login_channels.NewChannelStore(dbconn.GetWriteDbConn())
	if err := channelStore.Migrate(); err != nil {
		loggers.Logger.Fatal("渠道表结构初始化失败", zap.Error(err))
	}
	if err := channelStore.SeedDefault(); err != nil {
		loggers.Logger.Fatal("官方渠道种子初始化失败", zap.Error(err))
	}
	serverStore := login_servers_store.NewServerStore(dbconn.GetWriteDbConn())
	if err := serverStore.Migrate(); err != nil {
		loggers.Logger.Fatal("区服表结构初始化失败", zap.Error(err))
	}
	if err := serverStore.SeedIfEmpty(); err != nil {
		loggers.Logger.Fatal("区服种子初始化失败", zap.Error(err))
	}

	// 注入 handler 依赖
	gameClient = login_game_clients.NewGameClient()
	login_servers.LoginServerHandler.SetStore(accountStore, channelStore, serverStore, login_tokens.NewTokenManager(), gameClient)
}

func Shutdown() {
	if gameClient != nil {
		gameClient.Close()
	}
}
