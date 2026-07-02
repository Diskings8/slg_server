package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/services/internal/cores/cores_declarations"
)

type MarchInfo struct {
	rwLock          sync.RWMutex
	MarchID         cores_declarations.MarchID
	MarchType       cores_declarations.MarchType
	Team            *Team
	FromServerID    uint32
	ToServerID      uint32
	FromRoleID      uint64
	ToRoleID        uint64
	SrcFromMapID    uint32
	FromMapID       uint32
	ToMapID         uint32
	MarchState      cores_declarations.MarchState
	StartTimeUx     int64
	EndTimeUx       int64
	BaseEndTimeUx   int64
	FollowMarchID   cores_declarations.MarchID
	UnionID         uint32
	BaseMarchSpeed  uint32
	ActionUse       []cores_declarations.AnyThingUse
	PVPWinCount     uint32
	PVEWinCount     uint32
	VirtualData     uint64
	isVirtual       atomic.Bool
	isNeedSave      atomic.Bool
	isNeedDelete    atomic.Bool
	saving          atomic.Bool
	marchDoLocker   sync.Mutex
	AoiBlock        []cores_declarations.AoiScreen
	PassingAoiBlock []cores_declarations.AoiScreen
}

func (mi *MarchInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MarchInfo) Unlock() {
	mi.rwLock.Unlock()
}

func (mi *MarchInfo) LockMarchDo() bool {
	return mi.marchDoLocker.TryLock()
}

func (mi *MarchInfo) UnlockMarchDo() {
	mi.marchDoLocker.Unlock()
}

func (mi *MarchInfo) ClearUse() {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.ActionUse = []cores_declarations.AnyThingUse{}
}

func (mi *MarchInfo) AddAoiBlock(b cores_declarations.AoiScreen) {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.AoiBlock = append(mi.AoiBlock, b)
}

func (mi *MarchInfo) AddPassingAoiBlock(b cores_declarations.AoiScreen) {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
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

func (mi *MarchInfo) GetMarchType() cores_declarations.MarchType {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.MarchType
}

func (mi *MarchInfo) GetActionUse() []cores_declarations.AnyThingUse {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ActionUse
}

func (mi *MarchInfo) GetMarchStartAndEndTimeUx() (int64, int64) {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.StartTimeUx, mi.EndTimeUx
}

func (mi *MarchInfo) GetMapIDs() (uint32, uint32, uint32) {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.FromMapID, mi.ToMapID, mi.SrcFromMapID
}

func (mi *MarchInfo) GetFromMapID() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.FromMapID
}

func (mi *MarchInfo) GetToMapID() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ToMapID
}

func (mi *MarchInfo) GetSrcFromMapID() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.SrcFromMapID
}

func (mi *MarchInfo) GetMarchState() cores_declarations.MarchState {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.MarchState
}

func (mi *MarchInfo) GetFromRoleID() uint64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.FromRoleID
}

func (mi *MarchInfo) GetToRoleID() uint64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ToRoleID
}

func (mi *MarchInfo) GetFollowID() cores_declarations.MarchID {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.FollowMarchID
}

func (mi *MarchInfo) GetMarchTotalTimeUx() int64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.EndTimeUx - mi.StartTimeUx
}

func (mi *MarchInfo) GetStartTimeUx() int64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.StartTimeUx
}

func (mi *MarchInfo) GetEndTimeUx() int64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.EndTimeUx
}

func (mi *MarchInfo) GetBaseEndTimeUx() int64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.BaseEndTimeUx
}

func (mi *MarchInfo) GetFromServerID() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.FromServerID
}

func (mi *MarchInfo) GetToServerID() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ToServerID
}

func (mi *MarchInfo) GetTeam() *Team {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.Team
}

func (mi *MarchInfo) GetTotalWinCount() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.PVPWinCount + mi.PVEWinCount
}

func (mi *MarchInfo) GetPVPWinCount() uint32 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.PVPWinCount
}
