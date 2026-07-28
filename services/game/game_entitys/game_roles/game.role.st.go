package game_roles

import (
	"reflect"
	"strconv"
	"sync"

	"server.slg.com/api/protocol/pb/pb_gateway"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
)

var _ common_declarations.DataI = new(Role)

// Role 游戏角色
type Role struct {
	ID uint64 `json:"id,omitempty"`

	src      *Role         // 原始数据，用于获取副本数据
	copyLock *sync.RWMutex // 拷贝时使用
	isDelete bool          // 是否删除

	Status        int32                 `json:"status,omitempty"`          // 状态(1:正常,2:封号,3:禁言)
	StatusEndTime int64                 `json:"status_end_time,omitempty"` // 状态结束时间(单位:秒)
	BandStatus    int32                 `json:"band_status"`
	BandFinishUx  int64                 `json:"band_finish_ux"`
	GateID        uint32                `json:"-"`
	IP            string                `json:"-"`
	DeviceType    pb_gateway.DeviceType `json:"-"`
}

// UniqueID 唯一 id
func (r *Role) UniqueID() uint64 {
	return r.ID
}

// CacheKey 缓存键
func (r *Role) CacheKey() string {
	return strconv.FormatUint(r.ID, 10)
}

// Tag 实体类型名称，用于生成缓存 key
func (r *Role) Tag() string {
	return "game_role"
}

// TableName 数据库表名
func (r *Role) TableName() string {
	return "game_role"
}

// Save 保存到数据库
func (r *Role) Save(isDelete bool) error {
	if isDelete {
		return nil
	}
	return r.DBSave(nil)
}

// IsDelete 是否已标记为删除
func (r *Role) IsDelete() bool {
	return r.isDelete
}

// Marshal 编码，用于存入缓存
func (r *Role) Marshal() ([]byte, error) {
	return util_jsons.Marshal(r)
}

// Unmarshal 解码，从缓存中加载数据时使用
func (r *Role) Unmarshal(b []byte) error {
	r.New()
	err := util_jsons.Unmarshal(b, r)
	if err == nil {
		r.Init()
	}
	return err
}

// JSON2Bytes 获取存储的上一次编码数据
func (r *Role) JSON2Bytes() []byte {
	if b, ok := jsonCache.Get(r.CacheKey()); ok {
		return b.([]byte)
	}
	return nil
}

// Bytes2JSON 存储本次编码后的数据，用于下次比较是否有变化
func (r *Role) Bytes2JSON(b []byte) {
	if b == nil {
		jsonCache.Delete(r.CacheKey())
	} else {
		jsonCache.SetDefault(r.CacheKey(), b)
	}
}

// Init 初始化
// 遍历所有字段，对实现了 Init() 接口的子模块调用 Init
func (r *Role) Init() {
	r.GateID = 0

	val := reflect.ValueOf(r).Elem()
	for _, field := range val.Fields() {
		if !field.CanInterface() {
			continue
		}
		if module, ok := field.Interface().(interface{ Init() }); ok {
			module.Init()
		}
	}
}

// New 创建子模块数据
func (r *Role) New() {
	roleID := r.ID
	_ = roleID
	// 当存在子模块时在此处创建:
	// r.Attr = attr.New(roleID)
	// r.Heroes = heroes.New(roleID)
	// ...
}

// SetStatus 设置状态
func (r *Role) SetStatus(status int32, statusEndTime int64) {
	r.Status = status
	r.StatusEndTime = statusEndTime
}
