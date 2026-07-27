package game_roles

import (
	"context"
	"time"

	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/pollers"
)

// GetPollerMgr 获取角色数据轮询管理器
func GetPollerMgr() *pollers.PollerManager[*Role] {
	return pollerManager
}

// GetPoller 获取角色数据轮询器
func GetPoller(id uint64) (*pollers.Poller[*Role], error) {
	return pollerManager.Get(id)
}

// initPoller 初始化轮询管理器
func initPoller() {
	var (
		cacheSpec string
		dbSpec    string
		cacheTTL  time.Duration
	)

	if common_globals.IsDev() || common_globals.IsTest() {
		// 开发/测试环境:
		// 每隔 1 秒写入缓存
		// 每隔 10 秒写入数据库
		// 缓存中保存 6 小时
		cacheSpec = "*/1 * * * * *"
		dbSpec = "*/10 * * * * *"
		cacheTTL = time.Hour * 6
	} else {
		// 正式服:
		// 每隔 30 秒写入缓存
		// 每隔 1 分钟写入数据库
		// 缓存中保存 12 小时
		cacheSpec = "*/30 * * * * *"
		dbSpec = "0 */1 * * * *"
		cacheTTL = time.Hour * 12
	}

	pollerManager = pollers.New[*Role](
		context.Background(),
		loader,
		func() *Role { return &Role{} },
		cacheSpec,
		dbSpec,
		cacheTTL,
	)
}

// loader 从数据库加载角色数据
func loader(id uint64) (*Role, error) {
	r := &Role{ID: id}
	r.New()

	if !common_globals.IsTest() {
		readDB := dbconn.GetReadDbConn()
		if readDB == nil {
			// 若无独立读库则使用写库
			readDB = dbconn.GetWriteDbConn()
		}
		if err := r.DBGet(readDB); err != nil {
			return nil, err
		}
	}

	r.Init()
	return r, nil
}
