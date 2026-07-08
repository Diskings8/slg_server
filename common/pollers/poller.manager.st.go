package pollers

import (
	"sync"
	"time"

	"server.slg.com/common/utils/hashmaps"
)

type PollerManager[M DataI] struct {
	// all: 所有轮询器
	// key: 唯一id
	// value: 轮询器
	all hashmaps.Map[uint64, *Pool[M]]

	// lockers: id对应的锁
	// key: 唯一id
	// value: sync.Mutex
	lockers hashmaps.Map[uint64, *sync.Mutex]

	// loaderFunc 数据加载器,当本地数据和缓存数据都不存在时，加载数据
	loaderFunc LoaderFunc[M]

	// cacheTTL 远端缓存时长
	cacheTTL time.Duration

	//
	newFunc func() M

	//
	cacheQueueKey func() string

	//
	zeroVal M

	//
	asyncSave any

	//
	closeChan chan struct{}
}
