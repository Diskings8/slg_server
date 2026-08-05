package game_roles

import (
	"context"
	"time"

	"github.com/patrickmn/go-cache"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/pollers"
	"server.slg.com/common/utils/util_bytes"
	"server.slg.com/services/game/game_entitys/game_roles/cultivate_costs"
	"server.slg.com/services/game/game_entitys/game_roles/hero_skillcollections"
	"server.slg.com/services/game/game_entitys/game_roles/hero_skills"
	"server.slg.com/services/game/game_entitys/game_roles/role_attrs"
	"server.slg.com/services/game/game_entitys/game_roles/role_buildings"
	"server.slg.com/services/game/game_entitys/game_roles/role_formations"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_entitys/game_roles/role_items"
	"server.slg.com/services/game/game_entitys/game_roles/role_recruits"
)

var (
	// pollerManager 角色数据轮询管理器
	pollerManager *pollers.PollerManager[*Role]

	rolePool = util_bytes.NewPool(func() *Role {
		return &Role{}
	})

	// jsonCache 保存角色 []byte 数据，用于前后对比是否有变化
	// 如果有变化则存入 redis，如果没有变化则忽略，不存入 redis
	jsonCache = cache.New(time.Minute*10, time.Minute*5)
)

// get 从对象池获取角色
func get() *Role {
	return rolePool.Get()
}

// release 释放到对象池中
func release(r *Role) {
	rolePool.Put(r)
}

// GetPollerMgr 获取角色数据轮询管理器
func GetPollerMgr() *pollers.PollerManager[*Role] {
	return pollerManager
}

// getPoller 获取角色数据轮询器
func getPoller(id uint64) (*pollers.Poller[*Role], error) {
	return pollerManager.Get(id)
}

// Close 关闭轮询管理器
func Close() error {
	if pollerManager != nil {
		return pollerManager.Close()
	}
	return nil
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

// Init 初始化 game_roles 模块
func Init(writeDB common_declarations.DbcI) {
	//
	hero_skills.Init(writeDB)
	hero_skillcollections.Init(writeDB)
	cultivate_costs.Init(writeDB)
	role_heroes.Init(writeDB)
	role_items.Init(writeDB)
	role_buildings.Init(writeDB)
	role_formations.Init(writeDB)
	role_recruits.Init(writeDB)
	role_attrs.Init(writeDB)

	// 初始化轮询管理器
	initPoller()
}
