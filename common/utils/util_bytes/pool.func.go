package util_bytes

import (
	"sync"
	"time"
)

type ResetI interface {
	Reset()
}

func NewPool[T ResetI](f func() T) *Pool[T] {
	// 创建对象池
	pool := &Pool[T]{
		pool: sync.Pool{New: func() any { return f() }},
	}

	// 启动定时统计日志
	if IsStats {
		pool.done = make(chan bool)
		pool.ticker = time.NewTicker(5 * time.Minute) // 默认每分钟输出一次统计信息
		go func() {
			for {
				select {
				case <-pool.ticker.C:
					pool.LogStats()
				case <-pool.done:
					pool.ticker.Stop()
					return
				}
			}
		}()
	}

	return pool
}
