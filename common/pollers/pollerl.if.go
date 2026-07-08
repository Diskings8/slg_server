package pollers

import "sync"

// SaveEntityI 存储数据必须实现的接口
type SaveEntityI interface {
	UniqueID() uint64         // 唯一 id
	Tag() string              // 实体类型名称，会用于生成缓存 key
	Save(isDelete bool) error // 保存到数据库
	IsDelete() bool           // 是否已标记为删除
	Marshal() ([]byte, error) // 编码，从队列中存入缓存时用到
	Unmarshal([]byte) error   // 解码，从缓存中获取数据存入数据库过程中使用到
	JSONBytes() []byte        // 获取存储的上一次编码数据
	SetJSONBytes([]byte)      // 存储本次编码后的数据，用于下次比较是否有变化
}

// DataI 池内数据必须实现的接口
type DataI interface {
	SaveEntityI
	Copy(rw *sync.RWMutex) DataI
	IsCopy() bool
}
