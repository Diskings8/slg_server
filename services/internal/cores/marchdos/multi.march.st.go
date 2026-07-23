package marchdos

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

type MultiMarch struct {
	BaseMarch
	multi           []*marchs.MarchInfo
	markOff         int32
	marchLen        int
	arriveAfterFunc func(*map_managers.MapManager, []*marchs.MarchInfo)
}

func (m *MultiMarch) TryLock(marchLock, fromLock, toLock bool) bool {
	if marchLock {
		for inx, v := range m.multi {
			if v != nil {
				if v.TryLock() {
					m.markOff |= 1 << inx
					continue
				}
				m.unlock()
				return false
			}
		}
		m.marchLockOk = true
		m.markOff = 0
	}
	if fromLock {
		if !m.fromMapInfo.LockMarchDo() {
			m.unlock()
			return false
		}
		m.fromMapLockOk = true
	}
	if toLock {
		if !m.toMapInfo.LockMarchDo() {
			m.unlock()
			return false
		}
		m.toMapLockOk = true
	}
	return true
}

func (m *MultiMarch) unlock() {
	if m.marchLockOk || m.markOff != 0 {
		for i := 0; i < len(m.multi); i++ {
			if m.marchLockOk || m.markOff&1<<i != 0 {
				m.multi[i].Unlock()
			}
		}
		m.markOff = 0
		m.marchLockOk = false
	}
	if m.fromMapLockOk {
		m.fromMapInfo.UnlockMarchDo()
		m.fromMapLockOk = false
	}
	if m.toMapLockOk {
		m.toMapInfo.UnlockMarchDo()
		m.toMapLockOk = false
	}
}

func (m *MultiMarch) Init() {
	m.AddPrepareOpt(func(manager *map_managers.MapManager) {
		m.SetArriveAfterFunc(manager, m.multi)
	})
	m.AddDoOpt(func(manager *map_managers.MapManager) {})
	m.AddFinishOpt(func(manager *map_managers.MapManager) {})

	m.AddCallBackOpt(func(mgr *map_managers.MapManager) {
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			multiCallbackSwapDirection(mgr, info)
		}
	})

	m.AddCallBackNowOpt(func(mgr *map_managers.MapManager) {
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			multiCallbackNowInstantReturn(mgr, info)
		}
	})

	m.AddFinishBackArriveOpt(func(mgr *map_managers.MapManager) {
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			mgr.UpdateMarchPush(info)
		}
	})

	m.BaseMarch.Init()
}

// multiCallbackSwapDirection 单条行军召回核心（供 MultiMarch 复用）
func multiCallbackSwapDirection(mgr *map_managers.MapManager, info *marchs.MarchInfo) {
	state := info.MarchState
	if state == pb_maps_march.MarchState_Back ||
		state == pb_maps_march.MarchState_Error ||
		state == pb_maps_march.MarchState_Battle {
		return
	}

	oldToMapID := info.ToMapID

	now := time.Now().Unix()
	totalTime := info.EndTimeUx - info.StartTimeUx
	elapsed := now - info.StartTimeUx

	forwardTime := totalTime
	if elapsed > 0 && elapsed < totalTime {
		forwardTime = elapsed
	}
	info.EndTimeUx = now + forwardTime

	info.FromMapID = oldToMapID

	var returnTarget cores_declarations.MapID
	if info.TransitMapID >= 0 {
		returnTarget = info.TransitMapID
	} else {
		returnTarget = info.SrcFromMapID
	}
	info.ToMapID = returnTarget

	mgr.GetMarchManage().MapAttributeMarchCallBack(info)
	info.MarchState = pb_maps_march.MarchState_Back
	mgr.TickerAddMarch(info.MarchID, info.EndTimeUx)

	for _, v := range info.AoiBlock {
		v.MarchDelete(info)
	}
	for _, v := range info.PassingAoiBlock {
		v.PassingMarchDelete(info)
	}
	info.AoiBlock = nil
	info.PassingAoiBlock = nil
	mgr.MarchAOISetupSingle(info)
}

// multiCallbackNowInstantReturn 单条行军立即召回核心（供 MultiMarch 复用）
func multiCallbackNowInstantReturn(mgr *map_managers.MapManager, info *marchs.MarchInfo) {
	state := info.MarchState
	if state == pb_maps_march.MarchState_Back ||
		state == pb_maps_march.MarchState_Error ||
		state == pb_maps_march.MarchState_Battle {
		return
	}

	var returnTarget cores_declarations.MapID
	if info.TransitMapID >= 0 {
		returnTarget = info.TransitMapID
	} else {
		returnTarget = info.SrcFromMapID
	}
	info.ToMapID = returnTarget

	mgr.GetMarchManage().MapAttributeMarchCallBack(info)
	info.MarchState = pb_maps_march.MarchState_Back
	info.EndTimeUx = time.Now().Unix()
	mgr.TickerAddMarch(info.MarchID, info.EndTimeUx)

	for _, v := range info.AoiBlock {
		v.MarchDelete(info)
	}
	for _, v := range info.PassingAoiBlock {
		v.PassingMarchDelete(info)
	}
	info.AoiBlock = nil
	info.PassingAoiBlock = nil
	mgr.MarchAOISetupSingle(info)
}

func (m *MultiMarch) SetArriveAfterFunc(*map_managers.MapManager, []*marchs.MarchInfo) {}

// ---- MarchDoFuncHandleI 接口实现 ----

func (m *MultiMarch) Do() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	for _, info := range m.multi {
		if info == nil {
			continue
		}
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return m.BackArrive()
		}
		break
	}
	return m.BaseMarch.Do()
}

func (m *MultiMarch) BackArrive() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	if !m.TryLock(true, false, false) {
		return cores_declarations.ErrLockFailed
	}
	defer m.unlock()

	m.BaseMarch.BackArrive()

	for _, info := range m.multi {
		if info == nil {
			continue
		}
		m.mgr.UpdateMarchPush(info)
		m.mgr.GetMarchManage().DeleteMarch(info)
	}
	return nil
}

func (m *MultiMarch) ReTry() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Millisecond * 100)
		}
		if err := m.CallBack(); err == nil {
			return nil
		}
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			state := info.GetMarchState()
			if state == pb_maps_march.MarchState_Back ||
				state == pb_maps_march.MarchState_Error ||
				state == pb_maps_march.MarchState_Battle {
				return nil
			}
			break
		}
	}
	return cores_declarations.ErrLockFailed
}

func (m *MultiMarch) CallBackToSrcPoint() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	origTransits := make([]cores_declarations.MapID, len(m.multi))
	for i, info := range m.multi {
		if info != nil {
			origTransits[i] = info.TransitMapID
			info.TransitMapID = info.SrcFromMapID
		}
	}
	defer func() {
		for i, info := range m.multi {
			if info != nil {
				info.TransitMapID = origTransits[i]
			}
		}
	}()
	return m.CallBack()
}

func (m *MultiMarch) CallBackNowToSrcPoint() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	origTransits := make([]cores_declarations.MapID, len(m.multi))
	for i, info := range m.multi {
		if info != nil {
			origTransits[i] = info.TransitMapID
			info.TransitMapID = info.SrcFromMapID
		}
	}
	defer func() {
		for i, info := range m.multi {
			if info != nil {
				info.TransitMapID = origTransits[i]
			}
		}
	}()
	return m.CallBackNow()
}

func (m *MultiMarch) LockDo(marchLock, fromMapLock, toMapLock bool) error {
	if !m.TryLock(marchLock, fromMapLock, toMapLock) {
		return cores_declarations.ErrLockFailed
	}
	return nil
}

func (m *MultiMarch) CallBack() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	if !m.TryLock(true, false, false) {
		return cores_declarations.ErrLockFailed
	}
	defer m.unlock()

	m.BaseMarch.CallBack()

	for _, info := range m.multi {
		if info != nil {
			m.mgr.UpdateMarchPush(info)
		}
	}
	return nil
}

func (m *MultiMarch) CallBackNow() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	if !m.TryLock(true, false, false) {
		return cores_declarations.ErrLockFailed
	}
	defer m.unlock()

	m.BaseMarch.CallBackNow()

	for _, info := range m.multi {
		if info != nil {
			m.mgr.UpdateMarchPush(info)
		}
	}
	return nil
}

func (m *MultiMarch) Lock(marchDoLock, fromMapLock, toMapLock bool) bool {
	return m.TryLock(marchDoLock, fromMapLock, toMapLock)
}

func (m *MultiMarch) Unlock() {
	m.unlock()
}

func (m *MultiMarch) Leave() error {
	return nil
}
