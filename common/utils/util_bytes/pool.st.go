package util_bytes

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/utils/hashmaps"
)

// IsStats 是否开启未释放对象追踪统计
var IsStats = false

type Pool[T ResetI] struct {
	pool sync.Pool

	getLocations hashmaps.Map[string, *int64] // map[string]*int64
	putLocations hashmaps.Map[string, *int64] // map[string]*int64

	getCount atomic.Uint64
	putCount atomic.Uint64

	ticker *time.Ticker
	done   chan bool
}

func (p *Pool[T]) Put(v T) {
	// 一个非空接口值只要类型信息存在，即使值为 nil, v 也不会为 nil
	// TODO FIXME 性能
	if any(v) == nil || reflect.ValueOf(v).IsNil() {
		return
	}

	if IsStats {
		if common_globals.IsDev() {
			_, file, line, _ := runtime.Caller(1)
			key := fmt.Sprintf("%s:%d", file, line)
			count, _ := p.putLocations.LoadOrStore(key, new(int64))
			atomic.AddInt64(count, 1)
		}
		p.putCount.Add(1)
	}

	p.pool.Put(v)
}

func (p *Pool[T]) LogStats() {

}

func (p *Pool[T]) Get() T {
	v := p.pool.Get().(T)
	v.Reset()
	return v
}
