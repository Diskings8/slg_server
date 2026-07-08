package pollers

// Handle 数据操作句柄，使用完后必须 Release
type Poller[M DataI] struct {
	manager *Poller[M]
}

func (h *Handle[T]) Data() T {
	return h.data
}

func (h *Handle[T]) Release() {
	h.pool.release(h.id)
}
