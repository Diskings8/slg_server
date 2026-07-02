package packets

import "sync"

const (
	TcpHeaderSize     = 4 + 4 + 4 // 长度固定
	TcpLengthSizeTail = 4
	TcpSeqSizeTail    = 8
	TcpMsgIdSizeTail  = 12
)

var msgBufPool = sync.Pool{
	New: func() interface{} {
		// 这里按你的业务设置一个合理的初始大小，比如 4KB
		// 实际会根据需要扩容，但池里只会保存固定大小的对象
		return make([]byte, 4096)
	},
}

var headerBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, TcpHeaderSize)
	},
}
