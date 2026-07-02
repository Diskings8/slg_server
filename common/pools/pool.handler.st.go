package pools

// Handle 数据操作句柄，使用完后必须 Release
type Handle[T IHandler] struct {
	id   uint64
	data T
	pool *Pool[T]
}

func (h *Handle[T]) Data() T {
	return h.data
}

func (h *Handle[T]) Release() {
	h.pool.release(h.id)
}
