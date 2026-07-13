package util_bytes

var (
	binaryPool *BufferPool // buffer 内存池
	//reflectTypePool *TypePools  // 包含多种类型对象池
)

// Get 获取全局 buff 内存池
func Get() *BufferPool {
	return binaryPool
}
