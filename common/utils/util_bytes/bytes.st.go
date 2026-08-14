package util_bytes

import "sync"

var (
	binaryPool *BufferPool // buffer 内存池
	//reflectTypePool *TypePools  // 包含多种类型对象池
)

// Get 获取全局 buff 内存池
//
// 懒初始化：此前 binaryPool 从未被赋值，Get().Release(buffer) 在解引用 p.shards 时
// nil panic（roles poller 定时保存等路径触发）。Buffer 本就不从池取（每次新分配），
// Release 仅当 shards 中存在对应容量时才归还，空 shards map 即功能正确。
func Get() *BufferPool {
	if binaryPool == nil {
		binaryPool = &BufferPool{shards: map[int]*sync.Pool{}}
	}
	return binaryPool
}
