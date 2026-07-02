package marchs

import (
	"server.slg.com/services/internal/cores/mapdatas/info_maps"
)

type BaseMarch struct {
	marchManage   any                // 行军管理
	marchInfo     *MarchInfo         // 行军信息
	fromMapInfo   *info_maps.MapInfo // 来源地图信息
	toMapInfo     *info_maps.MapInfo // 目标地图信息
	marchLockOk   bool
	fromMapLockOk bool
	toMapLockOk   bool
}

func (m *BaseMarch) TryLock(marchLock, fromLock, toLock bool) bool {
	if marchLock {
		if !m.marchInfo.TryLock() {
			return false
		}
		m.marchLockOk = true
	}
	if fromLock {
		if !m.fromMapInfo.TryLock() {
			m.unlock()
			return false
		}
		m.fromMapLockOk = true
	}
	if toLock {
		if !m.toMapInfo.TryLock() {
			m.unlock()
			return false
		}
		m.toMapLockOk = true
	}
	return true
}

func (m *BaseMarch) unlock() {
	if m.marchLockOk {
		m.marchInfo.Unlock()
	}
	if m.fromMapLockOk {
		m.fromMapInfo.Unlock()
	}
	if m.toMapLockOk {
		m.toMapInfo.Unlock()
	}
}
