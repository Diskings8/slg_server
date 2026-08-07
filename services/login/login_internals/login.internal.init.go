package login_internals

import (
	"go.uber.org/zap"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_internals/login_game_clients"
	"server.slg.com/services/login/login_internals/login_servers_store"
	"server.slg.com/services/login/login_internals/login_tokens"
)

func Init() {
	db := dbconn.GetWriteDbConn()
	if err := login_accounts.Init(db); err != nil {
		loggers.Logger.Fatal("账号表结构初始化失败", zap.Error(err))
	}
	if err := login_channels.Init(db); err != nil {
		loggers.Logger.Fatal("渠道表结构初始化失败", zap.Error(err))
	}
	if err := login_servers_store.Init(db); err != nil {
		loggers.Logger.Fatal("区服表结构初始化失败", zap.Error(err))
	}

	// store/tokens/gameClient 全部包级单例，login_logics / LoginServer 直接 Get() 访问
	login_tokens.InitManager()
	login_game_clients.Init()
}

func Shutdown() {
	login_game_clients.Shutdown()
}
