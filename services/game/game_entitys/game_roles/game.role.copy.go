package game_roles

import (
	"sync"

	"server.slg.com/common/common_declarations"
)

// Copy 拷贝角色数据，返回副本
func (r *Role) Copy(rw *sync.RWMutex) common_declarations.DataI {
	v := Get()
	v.ID = r.ID
	v.src = r
	v.copyLock = rw
	v.GateID = r.GateID
	v.IP = r.IP
	v.DeviceType = r.DeviceType
	v.Status = r.Status
	v.StatusEndTime = r.StatusEndTime
	v.BandStatus = r.BandStatus
	v.BandFinishUx = r.BandFinishUx
	return v
}

// IsCopy 当前是否为拷贝的数据
func (r *Role) IsCopy() bool {
	return r.src != nil
}

// Reset 重置对象，用于对象池回收复用
func (r *Role) Reset() {
	r.ID = 0
	r.src = nil
	r.copyLock = nil
	r.isDelete = false
	r.Status = 0
	r.StatusEndTime = 0
	r.BandStatus = 0
	r.BandFinishUx = 0
	r.GateID = 0
	r.IP = ""
	r.DeviceType = 0
	r.Heroes = nil
	r.Skills = nil
	r.SkillCollections = nil
	r.CultivateCosts = nil
	r.Items = nil
	r.Buildings = nil
	r.Formations = nil
	r.Recruits = nil
}
