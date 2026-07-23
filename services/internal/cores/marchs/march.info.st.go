package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.MarchInfoI = (*MarchInfo)(nil)

// MarchInfo 行军信息，记录行军的完整状态，包括起止点、行军类型、时间、队伍、战报和 AOI 通行数据
type MarchInfo struct {
	RwLock          sync.RWMutex                     `gorm:"-"`
	MarchID         cores_declarations.MarchID       `gorm:"primaryKey;COMMENT:行军ID;"`
	MarchType       cores_declarations.MarchType     `gorm:"not null;COMMENT:行军类型;"`
	Team            *Team                            `gorm:"type:json;not null;COMMENT:部队数据;"`
	FromServerID    uint32                           `gorm:"not null;COMMENT:所属服务器;"`
	ToServerID      uint32                           `gorm:"not null;COMMENT:目标服务器;"`
	FromRoleID      uint64                           `gorm:"not null;COMMENT:归属者角色ID;"` // 当前归属者角色ID
	ExecRoleID      uint64                           `gorm:"not null;COMMENT:执行者角色ID;"` // 当前执行者角色ID
	SrcFromMapID    cores_declarations.MapID         `gorm:"not null;COMMENT:最开始的起始地图ID;"`
	TransitMapID    cores_declarations.MapID         `gorm:"default:-1;COMMENT:本次行军实际出发地（用于召回）；-1 时回退到 SrcFromMapID;"`
	FromMapID       cores_declarations.MapID         `gorm:"not null;COMMENT:当前行军起始地图ID;"`
	ToMapID         cores_declarations.MapID         `gorm:"not null;COMMENT:当前行军目标地图ID;"`
	MarchState      pb_maps_march.MarchState         `gorm:"not null;COMMENT:行军状态;"`
	StartTimeUx     int64                            `gorm:"not null;COMMENT:行军开始时间;"`
	EndTimeUx       int64                            `gorm:"not null;COMMENT:行军结束时间;"`
	FollowMarchID   cores_declarations.MarchID       `gorm:"not null;COMMENT:跟随的行军;"`
	UnionID         uint64                           `gorm:"not null;COMMENT:同盟ID;"`
	BaseMarchSpeed  uint32                           `gorm:"not null;COMMENT:基础行军速度;"`
	FinalMarchSpeed uint32                           `gorm:"not null;COMMENT:最后行军速度;"`
	ActionUse       []cores_declarations.AnyThingUse `gorm:"type:json;not null;COMMENT:行军消耗;"`
	Path            []cores_declarations.MapID       `gorm:"type:json;not null;COMMENT:路线;"`
	PVPWinCount     uint32                           `gorm:"not null;COMMENT:PVP连胜数量;"`
	PVEWinCount     uint32                           `gorm:"not null;COMMENT:PVE连胜数量;"`
	VirtualData     uint64                           `gorm:"not null;COMMENT:虚拟行军数据;"`
	isVirtual       bool                             `gorm:"not null;COMMENT:是否为虚拟行军;"`
	IsStay          bool                             `gorm:"not null;default:false;COMMENT:到达后停留;"`
	StayEndTimeUx   int64                            `gorm:"not null;default:0;COMMENT:停留结束时间;"`
	isNeedSave      atomic.Bool                      `gorm:"-"`
	isNeedDelete    atomic.Bool                      `gorm:"-"`
	isMock          atomic.Bool                      `gorm:"-"`
	saving          atomic.Bool                      `gorm:"-"`
	marchDoLocker   sync.Mutex                       `gorm:"-"`
	AoiBlock        []cores_declarations.AoiScreenI  `gorm:"-"`
	PassingAoiBlock []cores_declarations.AoiScreenI  `gorm:"-"`
}

func (mi *MarchInfo) GetRelocationVal() uint64 {
	var sum uint64
	for _, v := range mi.Team.Slots {
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			sum += uint64(v.GetHeroInfo().GetAttrRelocation().GetCurVal())
		}
	}
	return sum
}

func (mi *MarchInfo) TableName() string {
	return "MarchInfo"
}
func (mi *MarchInfo) AddPassingAOIBlock(i cores_declarations.AoiScreenI) {
	mi.RwLock.Lock()
	defer mi.RwLock.Unlock()
	mi.PassingAoiBlock = append(mi.PassingAoiBlock, i)
}

func (mi *MarchInfo) AddAOIBlock(i cores_declarations.AoiScreenI) {
	mi.RwLock.Lock()
	defer mi.RwLock.Unlock()
	mi.AoiBlock = append(mi.AoiBlock, i)
}

func (mi *MarchInfo) TryLock() bool {
	return mi.RwLock.TryLock()
}

func (mi *MarchInfo) Unlock() {
	mi.RwLock.Unlock()
}

func (mi *MarchInfo) LockMarchDo() bool {
	return mi.marchDoLocker.TryLock()
}

func (mi *MarchInfo) UnlockMarchDo() {
	mi.marchDoLocker.Unlock()
}

func (mi *MarchInfo) ClearUse() {
	mi.RwLock.Lock()
	defer mi.RwLock.Unlock()
	mi.ActionUse = []cores_declarations.AnyThingUse{}
}

func (mi *MarchInfo) AddAoiBlock(b cores_declarations.AoiScreenI) {
	mi.RwLock.Lock()
	defer mi.RwLock.Unlock()
	mi.AoiBlock = append(mi.AoiBlock, b)
}

func (mi *MarchInfo) AddPassingAoiBlock(b cores_declarations.AoiScreenI) {
	mi.RwLock.Lock()
	defer mi.RwLock.Unlock()
	mi.PassingAoiBlock = append(mi.PassingAoiBlock, b)
}

//------------------Is----------------//

func (mi *MarchInfo) IsVirtual() bool {
	return mi.isVirtual
}
func (mi *MarchInfo) IsMock() bool {
	return mi.isMock.Load()
}
func (mi *MarchInfo) IsNeedSave() bool {
	return mi.isNeedSave.Load()
}
func (mi *MarchInfo) IsNeedDelete() bool {
	return mi.isNeedDelete.Load()
}
func (mi *MarchInfo) IsSaving() bool {
	return mi.saving.Load()
}

func (mi *MarchInfo) IsMarchTypeAssist() bool {
	return mi.MarchType == cores_declarations.MarchTypeAssist
}

//--------------------------Get----------------------//

func (mi *MarchInfo) GetMarchID() cores_declarations.MarchID {
	return mi.MarchID
}

func (mi *MarchInfo) GetUnionID() uint64 {
	return mi.UnionID
}

func (mi *MarchInfo) GetActionUse() []cores_declarations.AnyThingUse {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.ActionUse
}

func (mi *MarchInfo) GetMarchStartAndEndTimeUx() (int64, int64) {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.StartTimeUx, mi.EndTimeUx
}

func (mi *MarchInfo) GetMapIDs() (cores_declarations.MapID, cores_declarations.MapID, cores_declarations.MapID) {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FromMapID, mi.ToMapID, mi.SrcFromMapID
}

func (mi *MarchInfo) GetFromMapID() cores_declarations.MapID {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FromMapID
}

func (mi *MarchInfo) GetToMapID() cores_declarations.MapID {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.ToMapID
}

func (mi *MarchInfo) GetSrcFromMapID() cores_declarations.MapID {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.SrcFromMapID
}

func (mi *MarchInfo) GetTransitMapID() cores_declarations.MapID {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.TransitMapID
}

func (mi *MarchInfo) GetMarchState() pb_maps_march.MarchState {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.MarchState
}

func (mi *MarchInfo) GetFromRoleID() uint64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FromRoleID
}

func (mi *MarchInfo) GetExecRoleID() uint64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.ExecRoleID
}

func (mi *MarchInfo) GetFollowID() cores_declarations.MarchID {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FollowMarchID
}

func (mi *MarchInfo) GetMarchTotalTimeUx() int64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.EndTimeUx - mi.StartTimeUx
}

func (mi *MarchInfo) GetStartTimeUx() int64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.StartTimeUx
}

func (mi *MarchInfo) GetEndTimeUx() int64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.EndTimeUx
}

func (mi *MarchInfo) GetFromServerID() uint32 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FromServerID
}

func (mi *MarchInfo) GetToServerID() uint32 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.ToServerID
}

func (mi *MarchInfo) GetTeam() *Team {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.Team
}

func (mi *MarchInfo) GetTotalWinCount() uint32 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.PVPWinCount + mi.PVEWinCount
}

func (mi *MarchInfo) GetPVPWinCount() uint32 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.PVPWinCount
}
