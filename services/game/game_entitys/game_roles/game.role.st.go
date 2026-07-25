package game_roles

import (
	"sync"

	"server.slg.com/api/protocol/pb/pb_gateway"
	"server.slg.com/common/common_declarations"
)

// Role 游戏角色
type Role struct {
	src          *Role // 原始数据，用于获取副本数据
	copyLock     *sync.RWMutex
	isDelete     bool                  // 是否删除
	BandStatus   int32                 `json:"band_status"`
	BandFinishUx int64                 `json:"band_finish_ux"`
	GateID       uint32                `json:"-"`
	IP           string                `json:"-"`
	DeviceType   pb_gateway.DeviceType `json:"-"`

	ID uint64 `json:"id,omitempty"`
}

func (r *Role) UniqueID() uint64 {
	return r.ID
}

func (r *Role) CacheKey() string {
	//TODO implement me
	panic("implement me")
}

func (r *Role) Tag() string {
	return "game_role"
}

func (r *Role) TableName() string {
	return "game_role"
}

func (r *Role) Save(isDelete bool) error {
	//TODO implement me
	panic("implement me")
}

func (r *Role) IsDelete() bool {
	return r.isDelete
}

func (r *Role) Marshal() ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Role) Unmarshal(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (r *Role) JSON2Bytes() []byte {
	//TODO implement me
	panic("implement me")
}

func (r *Role) Bytes2JSON(bytes []byte) {
	//TODO implement me
	panic("implement me")
}

func (r *Role) Copy(rw *sync.RWMutex) common_declarations.DataI {
	v := Get()
	v.ID = r.ID
	v.src = r
	v.copyLock = rw
	v.GateID = r.GateID
	v.IP = r.IP
	v.DeviceType = r.DeviceType
	return v
}

func (r *Role) IsCopy() bool {
	return r.src != nil
}

func (r *Role) Reset() {

}
