package roles

import (
	"slices"
	"strconv"
	"sync"
	"time"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ common_declarations.DataI = &Data{}

type Data struct {
	RoleID uint64 `gorm:"primaryKey;comment:角色ID"`
	// Queue 地图搜索私有队列（运行时缓存，非业务持久化字段）；map 类型 GORM 无法映射为列，
	// 标记 gorm:"-" 跳过，避免 role_data AutoMigrate/Save 失败
	Queue map[int32][]*GenerateQueue `gorm:"-"`
	// Brief 角色简略信息（嵌套 proto 结构，GORM 无法直接映射列）；标记 gorm:"-" 跳过。
	// loader 总是先 NewRoleDataInfo 初始化，故运行期 Brief 非 nil；跨重启不持久化，由 game 侧同步。
	Brief           *Brief `gorm:"-"`
	LastConnectTime int64  `gorm:"comment:最后连接时间"`
	copyLock        *sync.RWMutex
	src             *Data
}

func NewRoleDataInfo(id uint64) *Data {
	return &Data{
		RoleID:          id,
		Queue:           make(map[int32][]*GenerateQueue),
		Brief:           &Brief{RoleBrief: &pb_role.RoleBrief{}},
		LastConnectTime: time.Now().Unix(),
	}
}

func (d *Data) UniqueID() uint64 {
	return d.RoleID
}

func (d *Data) CacheKey() string {
	return strconv.FormatUint(d.RoleID, 10)
}

func (d *Data) Tag() string {
	return "role_data_info"
}

func (d *Data) TableName() string {
	return "role_data"
}

func (d *Data) IsDelete() bool {
	return false
}

func (d *Data) Marshal() ([]byte, error) {
	return util_jsons.Marshal(d)
}

func (d *Data) Unmarshal(b []byte) error {
	err := util_jsons.Unmarshal(b, d)
	if err == nil {
		d.Init()
	}
	return err
}

func (d *Data) JSON2Bytes() []byte {
	if b, ok := jsonCache.Get(d.CacheKey()); ok {
		return b.([]byte)
	}
	return nil
}

func (d *Data) Bytes2JSON(b []byte) {
	if b == nil {
		jsonCache.Delete(d.CacheKey())
	} else {
		jsonCache.SetDefault(d.CacheKey(), b)
	}
}

func (d *Data) Init() {
	return
}

func (d *Data) Reset() {
	d.RoleID = 0
	d.Queue = make(map[int32][]*GenerateQueue)
	d.Brief = &Brief{}
	d.src = nil
}

// AddQueue AddQueue
func (d *Data) AddQueue(queueKey int32, mapID cores_declarations.MapID) {
	queue := &GenerateQueue{
		MapID: mapID,
	}
	d.GetQueue()[queueKey] = append(d.GetQueue()[queueKey], queue)
}

// ReleaseRoleQueue 释放角色地图队列
func (d *Data) ReleaseRoleQueue(queueKey int32, baseMapInfo cores_declarations.MapID) {
	queues, ok := d.GetQueue()[queueKey]
	if !ok {
		return
	}
	d.GetQueue()[queueKey] = slices.DeleteFunc(queues, func(item *GenerateQueue) bool {
		return item.MapID == baseMapInfo
	})
}
