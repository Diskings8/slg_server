package marchdos

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

func NewSingleMarch() *SingleMarch {
	m := &SingleMarch{}
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

// ---- MarchDoFuncHandleI 接口实现 ----

// LockDo 尝试锁定行军和地块，失败返回 error
func (m *SingleMarch) LockDo(marchLock, fromMapLock, toMapLock bool) error {
	if !m.TryLock(marchLock, fromMapLock, toMapLock) {
		return cores_declarations.ErrLockFailed
	}
	return nil
}

// CallBack 召回行军
func (m *SingleMarch) CallBack() error {
	// TODO: 实现行军召回逻辑
	// 1. 切换行军状态为返回
	// 2. 重新计算返回路径和 AOI
	// 3. 推送更新
	return nil
}

// CallBackNow 立即召回行军
func (m *SingleMarch) CallBackNow() error {
	// TODO: 实现立即召回逻辑
	return nil
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
	// TODO: 行军离开后的清理逻辑
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
	m.BaseMarch.Init()
}
