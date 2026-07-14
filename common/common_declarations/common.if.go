package common_declarations

import (
	"sync"
	"sync/atomic"
)

type DbModelI interface {
	TableName() string
}

type DbcI interface {
	Error() error
	Table(tableName string) DbcI
	AutoMigrate(model DbModelI) error
	Find(any) DbcI
	Save(any) DbcI
	Where(query any, args ...any) DbcI
	Delete(query any, args ...any) DbcI
	Take(query any, args ...any) DbcI
	Create(any) DbcI
	CreateInBatches(march any, i int) DbcI
}

// SaveEntityI 存储数据必须实现的接口
type SaveEntityI interface {
	UniqueID() uint64         // 唯一 id
	CacheKey() string         // 缓存的key
	Tag() string              // 实体类型名称，会用于生成缓存 key
	TableName() string        // 数据库表名
	Save(isDelete bool) error // 保存到数据库
	IsDelete() bool           // 是否已标记为删除
	Marshal() ([]byte, error) // 编码，从队列中存入缓存时用到
	Unmarshal([]byte) error   // 解码，从缓存中获取数据存入数据库过程中使用到
	JSON2Bytes() []byte       // 获取存储的上一次编码数据
	Bytes2JSON([]byte)        // 存储本次编码后的数据，用于下次比较是否有变化
}

// DataI 池内数据必须实现的接口
type DataI interface {
	SaveEntityI
	Copy(rw *sync.RWMutex) DataI
	IsCopy() bool
}

// AsyncSaveEntityI 异步存储数据接口
type AsyncSaveEntityI interface {
	Tag() string          // 实体的唯一名，不同实体不可重复
	IsDelete() bool       // 是否已删除
	Saving() *atomic.Bool // 是否在保存中
	SaveDo()              // 保存处理函数
}
