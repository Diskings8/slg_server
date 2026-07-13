package pollers

import (
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
)

// Poller 数据操作句柄，使用完后必须 Release
type Poller[M common_declarations.DataI] struct {
	manager  *PollerManager[M]
	data     M
	rw       *sync.RWMutex
	ch       chan struct{}
	lockTime *atomic.Value // time.Time
	activeAt atomic.Int64  // 最后一次活跃时间戳
	tag      string        // 用于区分的标识
	stack    *atomic.Value // 最后一次引用, []byte
	saving   *atomic.Bool
}

func NewPoller[M common_declarations.DataI](manager *PollerManager[M], data M, tag string) *Poller[M] {
	p := &Poller[M]{
		manager:  manager,
		data:     data,
		tag:      tag,
		ch:       make(chan struct{}, 1),
		rw:       &sync.RWMutex{},
		saving:   &atomic.Bool{},
		stack:    &atomic.Value{},
		lockTime: &atomic.Value{},
	}
	p.lockTime.Store(time.Unix(0, 0))
	p.stack.Store([]byte{})
	p.active()
	p.Release()
	return p
}

func (p *Poller[M]) Get() (M, error) {
	return p.get()
}

// GetSync 获取数据, 等待，一直阻塞, 使用结束后需要调用 Release 释放
func (p *Poller[M]) GetSync() M {
	<-p.ch
	p.lockTime.Store(time.Now())
	p.active()
	return p.data
}

func (p *Poller[T]) Release() {
	select {
	case p.ch <- struct{}{}:
	default:
		// 并发情况处理
		loggers.Logger.Error("通道存在数据", zap.Uint64("id", p.ID()), zap.ByteString("Stack", p.stack.Load().([]byte)))
	}
}

func (p *Poller[M]) ID() uint64 {
	return p.data.UniqueID()
}

func (p *Poller[M]) Tag() string {
	return p.tag
}

// GetCopy 获取数据副本, 无需通过 Release 释放
func (p *Poller[M]) GetCopy() M {
	return p.data.Copy(p.rw).(M)
}

func (p *Poller[M]) active() {
	p.activeAt.Store(time.Now().Unix())

	if loggers.Logger.Core().Enabled(zap.WarnLevel) {
		p.stack.Store(debug.Stack())
	}
}

// get 获取游戏业务数据，直到超时
func (p *Poller[M]) get() (M, error) {
	t := time.NewTimer(common_declarations.PollerLockTimeout)
	defer t.Stop()

	select {
	case <-p.ch:
		p.lockTime.Store(time.Now())
		p.active()
		return p.data, nil
	case <-t.C:
		if p.lockTime.Load().(time.Time).Add(common_declarations.PollerLongLockTimeout).Before(time.Now()) {
			loggers.Logger.Error("锁定超过X秒 直接解锁", zap.Uint64("id", p.ID()), zap.ByteString("Stack", p.stack.Load().([]byte)))
			p.Release()
			p.lockTime.Store(time.Now())
			p.active()
			return p.data, nil
		}

		if loggers.Logger.Core().Enabled(zap.WarnLevel) {
			loggers.Logger.Error("Get TimeOut", zap.Uint64("id", p.ID()), zap.ByteString("Stack", p.stack.Load().([]byte)))
		}

		var zero M
		return zero, common_declarations.ErrPollerTimeout
	}
}
