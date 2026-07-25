package game_roles

import (
	"time"

	"github.com/patrickmn/go-cache"
	"server.slg.com/common/pollers"
	"server.slg.com/common/utils/util_bytes"
)

var (
	// pollers 角色数据轮询管理器
	pollerManager *pollers.PollerManager[*Role]

	rolePool = util_bytes.NewPool(func() *Role {
		return &Role{}
	})

	// jsonCache 保存角色 []byte 数据，用于前后对比是否有变化
	// 如果有变化则存入 redis，如果没有变化则忽略，不存入 redis
	jsonCache = cache.New(time.Minute*10, time.Minute*5)
)
