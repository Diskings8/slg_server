package util_bytes

import (
	"bytes"
	"sync"
)

type BufferPool struct {
	shards map[int]*sync.Pool
}

func (p *BufferPool) Buffer(n int) *bytes.Buffer {
	return bytes.NewBuffer(make([]byte, 0, n))
}

func (p *BufferPool) Release(b *bytes.Buffer) {
	if b != nil {
		if pool, ok := p.shards[b.Cap()]; ok {
			pool.Put(b)
		}
	}
}
