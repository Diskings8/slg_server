package marchdos

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

type MultiMarch struct {
	BaseMarch
	multi           []*marchs.MarchInfo
	markOff         int32
	marchLen        int
	MarchType       cores_declarations.MarchType
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
	m.BaseMarch.Init()
}

func (m *MultiMarch) SetArriveAfterFunc(*map_managers.MapManager, []*marchs.MarchInfo) {
}
