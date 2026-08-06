//go:build integration

package login_testutil

// 登录模块 DB 单测工具（集成测试，`go test -tags integration ./services/login/...` 运行）。
//
// 不依赖 sqlite：连接真实 mysql（DB.Common / common_db_0），连不上则跳过。
// 本地无 docker 时不带 tag 编译即完全跳过；docker 环境就绪后统一运行。

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_internals/login_servers"
	"server.slg.com/services/login/login_models"
)

var seq uint64

// UniqName 生成每次运行唯一的名称/ID 后缀，避免多次运行在同一 DB 中残留冲突
func UniqName(base string) string {
	return fmt.Sprintf("%s_%d_%d", base, time.Now().UnixNano(), atomic.AddUint64(&seq, 1))
}

// InitDB 初始化配置 + 写库连接；mysql 不可达则跳过测试
func InitDB(t *testing.T) {
	t.Helper()
	loggers.Init()
	common_configs.LoadByFormat("yaml", common_globals.GetEnvPath())

	dsn := common_configs.GetConf().DB.Common.Dsn()
	if err := dbconn.InitDB("mysql", dsn, dsn); err != nil {
		t.Skipf("mysql 不可达（%v），跳过 DB 单测；docker 环境就绪后统一运行", err)
	}
}

// SetupStores 初始化三张账号域表并返回 store（已 Migrate + 渠道/区服种子）
func SetupStores(t *testing.T) (*login_accounts.AccountStore, *login_channels.ChannelStore, *login_servers.ServerStore) {
	t.Helper()
	InitDB(t)

	db := dbconn.GetWriteDbConn()

	accStore := login_accounts.NewAccountStore(db)
	if err := accStore.Migrate(); err != nil {
		t.Fatalf("migrate account: %v", err)
	}

	chStore := login_channels.NewChannelStore(db)
	if err := chStore.Migrate(); err != nil {
		t.Fatalf("migrate channel: %v", err)
	}
	if err := chStore.SeedDefault(); err != nil {
		t.Fatalf("seed official channel: %v", err)
	}
	// 声明第三方渠道（类型 1，供跨渠道绑定用例）
	if ch, err := chStore.GetChannel(1); err != nil {
		t.Fatalf("query third-party channel: %v", err)
	} else if ch == nil {
		if err := db.Create(&login_models.Channel{ChannelType: 1, ChannelName: "测试渠道", Status: 0}).Error(); err != nil {
			t.Fatalf("declare third-party channel: %v", err)
		}
	}

	svStore := login_servers.NewServerStore(db)
	if err := svStore.Migrate(); err != nil {
		t.Fatalf("migrate server: %v", err)
	}
	if err := svStore.SeedIfEmpty(); err != nil {
		t.Fatalf("seed server: %v", err)
	}

	return accStore, chStore, svStore
}
