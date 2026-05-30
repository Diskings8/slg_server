package packets

type Packet struct {
	Length uint32
	Seq    uint32
	MsgID  uint32
	Body   []byte
}

func (p *Packet) Release() {
	msgBufPool.Put(p.Body)
	p.Body = nil
}

func GetMsgBuf(size int) []byte {
	buf := msgBufPool.Get().([]byte)
	if cap(buf) < size {
		// 如果容量不够，扩容
		buf = make([]byte, size)
	} else {
		// 裁剪到需要的长度，避免写入时超出
		buf = buf[:size]
	}
	return buf
}

func PutMsgBuf(buf []byte) {
	msgBufPool.Put(buf[:cap(buf)]) // 归还完整容量的slice
}

func GetHeadBuf(size int) []byte {
	buf := headerBufPool.Get().([]byte)
	if cap(buf) < size {
		// 如果容量不够，扩容
		buf = make([]byte, size)
	} else {
		// 裁剪到需要的长度，避免写入时超出
		buf = buf[:size]
	}
	return buf
}

func PutHeadBuf(buf []byte) {
	// 重置缓冲区，避免脏数据
	for i := range buf {
		buf[i] = 0
	}
	headerBufPool.Put(buf[:cap(buf)]) // 归还完整容量的slice
}
