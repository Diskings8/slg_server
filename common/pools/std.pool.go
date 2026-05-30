package pools

import (
	"sync"
	"sync/atomic"
)

type entry[T IHandler] struct {
	handler  T
	refCount int32
}

// Pool 泛型 handler 池，线程安全
type Pool[T IHandler] struct {
	mu    sync.RWMutex
	items map[uint64]*entry[T]
}

func New[T IHandler]() *Pool[T] {
	return &Pool[T]{
		items: make(map[uint64]*entry[T]),
	}
}

// Add 添加或覆盖一个 handler
func (p *Pool[T]) Add(handler T) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[handler.ID()] = &entry[T]{handler: handler}
}

// Get 根据 id 获取操作句柄，返回 nil, false 表示不存在
func (p *Pool[T]) Get(id uint64) (*Handle[T], bool) {
	p.mu.RLock()
	e, ok := p.items[id]
	p.mu.RUnlock()
	if !ok {
		return nil, false
	}
	atomic.AddInt32(&e.refCount, 1)
	return &Handle[T]{id: id, data: e.handler, pool: p}, true
}

// Remove 从池中删除指定 handler
func (p *Pool[T]) Remove(id uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, id)
}

// Len 返回池中 handler 数量
func (p *Pool[T]) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.items)
}

// Range 遍历所有 handler
func (p *Pool[T]) Range(fn func(id uint64, handler T) bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	for id, e := range p.items {
		if !fn(id, e.handler) {
			break
		}
	}
}

func (p *Pool[T]) release(id uint64) {
	p.mu.RLock()
	e, ok := p.items[id]
	p.mu.RUnlock()
	if !ok {
		return
	}
	atomic.AddInt32(&e.refCount, -1)
}
