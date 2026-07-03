package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.MarchInfoI = (*MarchInfo)(nil)

// MarchInfo 行军信息，记录行军的完整状态，包括起止点、行军类型、时间、队伍、战报和 AOI 通行数据
type MarchInfo struct {
	RwLock          sync.RWMutex
	MarchID         cores_declarations.MarchID
	Team            *Team
	FromServerID    uint32
	ToServerID      uint32
	FromRoleID      uint64
	ToRoleID        uint64
	SrcFromMapID    cores_declarations.MapID
	FromMapID       cores_declarations.MapID
	ToMapID         cores_declarations.MapID
	MarchState      cores_declarations.MarchState
	StartTimeUx     int64
	EndTimeUx       int64
	BaseEndTimeUx   int64
	FollowMarchID   cores_declarations.MarchID
	UnionID         uint32
	BaseMarchSpeed  uint32
	ActionUse       []cores_declarations.AnyThingUse
	Path            []cores_declarations.MarchID `gorm:"type:json;serializer:json;not null;COMMENT:路线;"`
	PVPWinCount     uint32
	PVEWinCount     uint32
	VirtualData     uint64
	isVirtual       atomic.Bool
	isNeedSave      atomic.Bool
	isNeedDelete    atomic.Bool
	saving          atomic.Bool
	marchDoLocker   sync.Mutex
	AoiBlock        []cores_declarations.AoiScreenI
	PassingAoiBlock []cores_declarations.AoiScreenI
}

func (mi *MarchInfo) TableName() string {
	//TODO implement me
	panic("implement me")
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
	return mi.isVirtual.Load()
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

//--------------------------Get----------------------//

func (mi *MarchInfo) GetMarchID() cores_declarations.MarchID {
	return mi.MarchID
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

func (mi *MarchInfo) GetMarchState() cores_declarations.MarchState {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.MarchState
}

func (mi *MarchInfo) GetFromRoleID() uint64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.FromRoleID
}

func (mi *MarchInfo) GetToRoleID() uint64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.ToRoleID
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

func (mi *MarchInfo) GetBaseEndTimeUx() int64 {
	mi.RwLock.RLock()
	defer mi.RwLock.RUnlock()
	return mi.BaseEndTimeUx
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
