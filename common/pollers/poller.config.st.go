package pollers

import "time"

// PoolConf 缓存池持久化配置，控制缓存写入和数据库写入的时间间隔
type PoolConf struct {
	SaveCacheDuration time.Duration
	SaveDbDuration    time.Duration
}
