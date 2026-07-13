package pollers

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/utils/asyncsave_entity"
	"server.slg.com/common/utils/crontabs"
)

func New[M common_declarations.DataI](ctx context.Context, loaderF common_declarations.LoaderFunc[M], newF func() M, cacheSpec, dbSpec string, cacheTTL time.Duration) *PollerManager[M] {
	tag := (*new(M)).Tag()
	if loaderF == nil {
		panic(fmt.Sprintf("loader cannot be nil, tag: %s", tag))
	}

	ase, err := asyncsave_entity.NewAsyncSaveEntity(cacheSpec, tag)
	if err != nil {
		panic(err)
	}

	pollerManager := &PollerManager[M]{
		ctx:        ctx,
		loaderFunc: loaderF,
		cacheTTL:   cacheTTL,
		newFunc:    newF,
		asyncSave:  ase,
		closeChan:  make(chan struct{}),
		cacheQueueKey: sync.OnceValue(func() string {
			return cacheconn.Key(common_declarations.CaCheQueueKEY, tag)
		}),
	}

	if pollerManager.dbCronEntryID, err = crontabs.AddNotRaceFunc(dbSpec, pollerManager.saveToDB); err != nil {
		panic(err)
	}

	go pollerManager.cleanup()

	return pollerManager
}

// Close 关闭
func (p *PollerManager[M]) Close() error {
	crontabs.Get().Remove(p.dbCronEntryID)

	asyncsave_entity.RemoveAsyncSave(p.zeroVal.Tag())

	close(p.closeChan)
	return nil
}

func (p *PollerManager[M]) Get(id uint64) (*Poller[M], error) {
	if id < 1 {
		return nil, errors.New("invalid id")
	}

	// 从当前管理器中查找
	value, found := p.all.Load(id)
	if found {
		return value, nil
	}

	locker, _ := p.lockers.LoadOrStore(id, &sync.Mutex{})
	if !locker.TryLock() {
		maxRetries := 10
		retryCount := 0
		for !locker.TryLock() {
			if retryCount >= maxRetries {
				return nil, fmt.Errorf("failed to acquire poller lock: timeout, tag: %s", p.zeroVal.Tag())
			}
			time.Sleep(50 * time.Millisecond)
			retryCount++
		}
	}
	defer locker.Unlock()

	// 再次检查是否已加载
	value, found = p.all.Load(id)
	if found {
		return value, nil
	}

	// 加载数据
	data, err := p.load(id)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w, tag: %s", err, p.zeroVal.Tag())
	}

	// 创建轮询器
	poller := NewPoller(p, data, p.zeroVal.Tag())
	p.all.Store(id, poller)

	return poller, nil
}
