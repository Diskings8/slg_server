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

	// ---- 默认召回逻辑：遍历 multi，对每个行军执行方向反转 ----
	m.AddCallBackOpt(func(mgr *map_managers.MapManager) {
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			multiCallbackSwapDirection(mgr, info)
		}
	})

	// ---- 默认立即召回逻辑 ----
	m.AddCallBackNowOpt(func(mgr *map_managers.MapManager) {
		for _, info := range m.multi {
			if info == nil {
				continue
			}
			multiCallbackNowInstantReturn(mgr, info)
		}
	})

	// ---- 默认召回到达逻辑 ----
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
//
// 注意：调用方需已持有对应 MarchInfo 的写锁，直接访问字段而非通过 getter。
func multiCallbackSwapDirection(mgr *map_managers.MapManager, info *marchs.MarchInfo) {
	state := info.MarchState
	if state == pb_maps_march.MarchState_Back ||
		state == pb_maps_march.MarchState_Error ||
		state == pb_maps_march.MarchState_Battle {
		return
	}

	oldFromMapID := info.FromMapID
	oldToMapID := info.ToMapID

	now := time.Now().Unix()
	totalTime := info.EndTimeUx - info.StartTimeUx
	elapsed := now - info.StartTimeUx

	var returnEndTime int64
	switch {
	case elapsed <= 0:
		returnEndTime = now + totalTime
	case elapsed >= totalTime:
		returnEndTime = now + totalTime
	default:
		returnEndTime = now + elapsed
	}
	info.EndTimeUx = returnEndTime

	info.FromMapID = oldToMapID
	// 返回目标：优先使用 TransitMapID（实际出发地），回退到 SrcFromMapID（初始主城）
	var returnTarget cores_declarations.MapID
	if info.TransitMapID >= 0 {
		returnTarget = info.TransitMapID
	} else {
		returnTarget = info.SrcFromMapID
	}
	info.ToMapID = returnTarget

	// 更新 MapAttribute（方法内部直接访问字段，避免已持有写锁时 RLock 死锁）
	mgr.GetMarchManage().MapAttributeMarchCallBack(info)

	info.MarchState = pb_maps_march.MarchState_Back
	mgr.TickerAddMarch(info.MarchID, returnEndTime)

	// AOI 路径重算：清除旧路径 AOI，重新计算返回路径
	// 注意：直接访问字段，避免已持有写锁时 RLock 死锁
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
//
// 注意：调用方需已持有对应 MarchInfo 的写锁，直接访问字段而非通过 getter。
func multiCallbackNowInstantReturn(mgr *map_managers.MapManager, info *marchs.MarchInfo) {
	state := info.MarchState
	if state == pb_maps_march.MarchState_Back ||
		state == pb_maps_march.MarchState_Error ||
		state == pb_maps_march.MarchState_Battle {
		return
	}

	oldFromMapID := info.FromMapID
	oldToMapID := info.ToMapID

	info.FromMapID = oldToMapID
	// 返回目标：优先使用 TransitMapID，回退到 SrcFromMapID
	var nowReturnTarget cores_declarations.MapID
	if info.TransitMapID >= 0 {
		nowReturnTarget = info.TransitMapID
	} else {
		nowReturnTarget = info.SrcFromMapID
	}
	info.ToMapID = nowReturnTarget

	// 更新 MapAttribute（方法内部直接访问字段，避免已持有写锁时 RLock 死锁）
	mgr.GetMarchManage().MapAttributeMarchCallBack(info)
	info.MarchState = pb_maps_march.MarchState_Back
	info.EndTimeUx = time.Now().Unix()
	mgr.TickerAddMarch(info.MarchID, info.EndTimeUx)

	// AOI 路径重算：清除旧路径 AOI，重新计算返回路径
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

func (m *MultiMarch) SetArriveAfterFunc(*map_managers.MapManager, []*marchs.MarchInfo) {
}

// ---- MarchDoFuncHandleI 接口实现 ----

// Do 执行行军到达处理
//
// 根据行军状态分流：
//   - MarchState_Back → 召回到达处理（BackArrive）
//   - 其他状态 → 正常到达处理（base Do 流程）
func (m *MultiMarch) Do() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	// 检查第一个非空行军的状态来分流
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

// BackArrive 召回到达处理
//
// 当 MultiMarch 在 Back 状态下到达原出发地时调用。
// 遍历所有行军，推送最终状态后逐一删除。
func (m *MultiMarch) BackArrive() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	if !m.TryLock(true, false, false) {
		return cores_declarations.ErrLockFailed
	}
	defer m.unlock()

	m.BaseMarch.BackArrive()

	// 推送并删除每个行军
	for _, info := range m.multi {
		if info == nil {
			continue
		}
		m.mgr.UpdateMarchPush(info)
		m.mgr.GetMarchManage().DeleteMarch(info)
	}
	return nil
}

// ReTry 召回重试
//
// 当 MultiMarch 的 CallBack 因锁竞争等临时原因失败时，进行有限次重试。
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
		// 检查第一个非空行军是否已进入不可召回状态
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

// CallBackToSrcPoint 强制召回所有行军到 SrcFromMapID
//
// 无视 TransitMapID（各自的实际出发地），直接回到各自的 SrcFromMapID（最初起始点）。
func (m *MultiMarch) CallBackToSrcPoint() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	// 临时覆盖所有行军的 TransitMapID 为 SrcFromMapID
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

// CallBackNowToSrcPoint 强制立即召回所有行军到 SrcFromMapID
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

// LockDo 尝试锁定所有行军和地块
func (m *MultiMarch) LockDo(marchLock, fromMapLock, toMapLock bool) error {
	if !m.TryLock(marchLock, fromMapLock, toMapLock) {
		return cores_declarations.ErrLockFailed
	}
	return nil
}

// CallBack 召回所有行军
func (m *MultiMarch) CallBack() error {
	if len(m.multi) == 0 || m.mgr == nil {
		return nil
	}
	// 尝试锁定所有行军
	if !m.TryLock(true, false, false) {
		return cores_declarations.ErrLockFailed
	}
	defer m.unlock()

	m.BaseMarch.CallBack()

	// 分别推送每个行军
	for _, info := range m.multi {
		if info != nil {
			m.mgr.UpdateMarchPush(info)
		}
	}
	return nil
}

// CallBackNow 立即召回所有行军
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

// Lock 尝试锁定所有行军和地块
func (m *MultiMarch) Lock(marchDoLock, fromMapLock, toMapLock bool) bool {
	return m.TryLock(marchDoLock, fromMapLock, toMapLock)
}

// Unlock 解锁所有行军和地块
func (m *MultiMarch) Unlock() {
	m.unlock()
}

// Leave 行军离开时的清理
func (m *MultiMarch) Leave() error {
	return nil
}
