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
	RoleID          uint64
	Queue           map[int32][]*GenerateQueue
	Brief           *Brief
	LastConnectTime int64
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
