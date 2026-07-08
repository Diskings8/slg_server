package pollers

import (
	"fmt"
	"sync"
	"time"

	"server.slg.com/common/conns/cacheconn"
)

func New[M DataI](loaderF LoaderFunc[M], newF func() M, cacheSpec, dbSpec string, cacheTTL time.Duration) *PollerManager[M] {
	tag := (*new(M)).Tag()
	if loaderF == nil {
		panic(fmt.Sprintf("loader cannot be nil, tag: %s", tag))
	}

	// todo asyncSave

	pollerManage := &PollerManager[M]{
		loaderFunc: loaderF,
		cacheTTL:   cacheTTL,
		newFunc:    newF,
		asyncSave:  nil,
		closeChan:  make(chan struct{}),
		cacheQueueKey: sync.OnceValue(func() string {
			return cacheconn.CacheManager.Key(CACHEQUEUEKEY, tag)
		}),
	}
	return pollerManage
}
