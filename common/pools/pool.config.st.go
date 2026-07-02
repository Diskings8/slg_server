package pools

import "time"

type PoolConf struct {
	SaveCacheDuration time.Duration
	SaveDbDuration    time.Duration
}
