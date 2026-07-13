package util_bytes

import "bytes"

type BufferPool struct {
}

func (p *BufferPool) Buffer(n int) *bytes.Buffer {
	return bytes.NewBuffer(make([]byte, 0, n))
}

func (p *BufferPool) Release(b *bytes.Buffer) {
	return
}
