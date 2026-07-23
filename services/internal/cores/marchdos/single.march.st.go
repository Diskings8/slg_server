package marchdos

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

func NewSingleMarch(mm *map_managers.MapManager) *SingleMarch {
	m := &SingleMarch{}
	m.SetBase(mm)
	m.Init()
	return m
}

type SingleMarch struct {
	BaseMarch
	single          *marchs.MarchInfo
	arriveAfterFunc func(*map_managers.MapManager, *marchs.MarchInfo)
}

// SetMarchInfo 设置行军信息
func (m *SingleMarch) SetMarchInfo(info *marchs.MarchInfo) {
	m.single = info
}

// SetMarchManage 设置行军管理器
func (m *SingleMarch) SetMarchManage(manage *marchs.MarchInfoManager) {
	m.marchManage = manage
}

// SetFromMapInfo 设置来源地块
func (m *SingleMarch) SetFromMapInfo(info *map_datas.MapInfo) {
	m.fromMapInfo = info
}

// SetToMapInfo 设置目标地块
func (m *SingleMarch) SetToMapInfo(info *map_datas.MapInfo) {
	m.toMapInfo = info
}

// SetArriveAfterFunc 设置到达后回调
func (m *SingleMarch) SetArriveAfterFunc(f func(*map_managers.MapManager, *marchs.MarchInfo)) {
	m.arriveAfterFunc = f
}

// MarchInfo 返回行军信息
func (m *SingleMarch) MarchInfo() *marchs.MarchInfo {
	return m.single
}

// GetFromMapInfo 返回来源地块
func (m *SingleMarch) GetFromMapInfo() *map_datas.MapInfo {
	return m.fromMapInfo
}

// GetToMapInfo 返回目标地块
func (m *SingleMarch) GetToMapInfo() *map_datas.MapInfo {
	return m.toMapInfo
}

// ---- MarchDoFuncHandleI 接口实现 ----

// LockDo 尝试锁定行军和地块，失败返回 error
func (m *SingleMarch) LockDo(marchLock, fromMapLock, toMapLock bool) error {
	if !m.TryLock(marchLock, fromMapLock, toMapLock) {
		return cores_declarations.ErrLockFailed
	}
	return nil
}

// Do 执行行军到达处理
//
// 根据行军状态分流：
//   - MarchState_Back → 召回到达处理（BackArrive）
//   - 其他状态 → 正常到达处理（base Do 流程）
func (m *SingleMarch) Do() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	if m.single.GetMarchState() == pb_maps_march.MarchState_Back {
		return m.BackArrive()
	}
	return m.BaseMarch.Do()
}

// CallBack 召回行军
//
// 锁定行军 → 执行回调链 → 推送 → 持久化 → 解锁。
func (m *SingleMarch) CallBack() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	if !m.single.TryLock() {
		return cores_declarations.ErrLockFailed
	}
	defer m.single.Unlock()

	m.BaseMarch.CallBack()

	m.mgr.UpdateMarchPush(m.single)
	m.mgr.GetMarchManage().Save(m.single)
	return nil
}

// CallBackNow 立即召回行军
func (m *SingleMarch) CallBackNow() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	if !m.single.TryLock() {
		return cores_declarations.ErrLockFailed
	}
	defer m.single.Unlock()

	m.BaseMarch.CallBackNow()

	m.mgr.UpdateMarchPush(m.single)
	m.mgr.GetMarchManage().Save(m.single)
	return nil
}

// BackArrive 召回到达处理
func (m *SingleMarch) BackArrive() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	if !m.single.TryLock() {
		return cores_declarations.ErrLockFailed
	}
	defer m.single.Unlock()

	m.BaseMarch.BackArrive()

	m.mgr.UpdateMarchPush(m.single)
	m.mgr.GetMarchManage().DeleteMarch(m.single)
	return nil
}

// ReTry 召回重试
func (m *SingleMarch) ReTry() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Millisecond * 100)
		}
		if err := m.CallBack(); err == nil {
			return nil
		}
		state := m.single.GetMarchState()
		if state == pb_maps_march.MarchState_Back ||
			state == pb_maps_march.MarchState_Error ||
			state == pb_maps_march.MarchState_Battle {
			return nil
		}
	}
	return cores_declarations.ErrLockFailed
}

// CallBackToSrcPoint 强制召回行军到 SrcFromMapID
func (m *SingleMarch) CallBackToSrcPoint() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	origTransit := m.single.TransitMapID
	m.single.TransitMapID = m.single.SrcFromMapID
	defer func() { m.single.TransitMapID = origTransit }()
	return m.CallBack()
}

// CallBackNowToSrcPoint 强制立即召回行军到 SrcFromMapID
func (m *SingleMarch) CallBackNowToSrcPoint() error {
	if m.single == nil || m.mgr == nil {
		return nil
	}
	origTransit := m.single.TransitMapID
	m.single.TransitMapID = m.single.SrcFromMapID
	defer func() { m.single.TransitMapID = origTransit }()
	return m.CallBackNow()
}

// Lock 尝试锁定行军和地块（返回 bool）
func (m *SingleMarch) Lock(marchDoLock, fromMapLock, toMapLock bool) bool {
	return m.TryLock(marchDoLock, fromMapLock, toMapLock)
}

// Unlock 解锁行军和地块
func (m *SingleMarch) Unlock() {
	m.unlock()
}

// Leave 行军离开时的清理
func (m *SingleMarch) Leave() error {
	return nil
}

func (m *SingleMarch) TryLock(marchLock, fromLock, toLock bool) bool {
	if marchLock {
		if !m.single.TryLock() {
			m.unlock()
			return false
		}
		m.marchLockOk = true
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

func (m *SingleMarch) unlock() {
	if m.marchLockOk {
		m.single.Unlock()
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

func (m *SingleMarch) Init() {
	m.AddPrepareOpt(func(manager *map_managers.MapManager) {})
	m.AddDoOpt(func(manager *map_managers.MapManager) {})
	m.AddFinishOpt(func(manager *map_managers.MapManager) {})

	// ---- 默认召回逻辑 ----
	m.AddCallBackOpt(func(mgr *map_managers.MapManager) {
		m.callbackSwapDirection(mgr)
	})

	// ---- 默认立即召回逻辑 ----
	m.AddCallBackNowOpt(func(mgr *map_managers.MapManager) {
		m.callbackNowInstantReturn(mgr)
	})

	// ---- 默认召回到达逻辑 ----
	m.AddFinishBackArriveOpt(func(mgr *map_managers.MapManager) {
		if m.single == nil {
			return
		}
		mgr.UpdateMarchPush(m.single)
	})

	m.BaseMarch.Init()
}

// callbackSwapDirection 召回核心：反转行军方向、重算时间、更新状态
//
// 供默认 callBackOpt 使用。要求在 m.single 已加锁状态下调用。
// 注意：已持有写锁，直接访问字段而非通过 getter（getter 会再次 RLock 导致死锁）。
func (m *SingleMarch) callbackSwapDirection(mgr *map_managers.MapManager) {
	info := m.single
	if info == nil {
		return
	}

	state := info.MarchState
	if state == pb_maps_march.MarchState_Back ||
		state == pb_maps_march.MarchState_Error ||
		state == pb_maps_march.MarchState_Battle {
		return
	}

	oldToMapID := info.ToMapID

	// 重算返回时间：等比例对称，走了多久就需要多久返回
	now := time.Now().Unix()
	totalTime := info.EndTimeUx - info.StartTimeUx
	elapsed := now - info.StartTimeUx

	forwardTime := totalTime
	if elapsed > 0 && elapsed < totalTime {
		forwardTime = elapsed
	}
	info.EndTimeUx = now + forwardTime

	// 交换方向
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

	// AOI 路径重算
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

// callbackNowInstantReturn 立即召回核心：方向反转 + 时间归零
//
// EndTimeUx 设为当前时间，tick 立即触发到达处理。
func (m *SingleMarch) callbackNowInstantReturn(mgr *map_managers.MapManager) {
	info := m.single
	if info == nil {
		return
	}

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

	// AOI 路径重算
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
