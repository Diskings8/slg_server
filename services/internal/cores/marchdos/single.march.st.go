package marchdos

import (
	"server.slg.com/services/internal/cores/cores_declarations"
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
	MarchType       cores_declarations.MarchType
	arriveAfterFunc func(*map_managers.MapManager, *marchs.MarchInfo)
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

func (m *SingleMarch) unlock() {
	if m.marchLockOk {
		m.single.Unlock()
		m.marchLockOk = false
	}
	if m.fromMapLockOk {
		m.fromMapInfo.Unlock()
		m.fromMapLockOk = false
	}
	if m.toMapLockOk {
		m.toMapInfo.Unlock()
		m.toMapLockOk = false
	}
}

func (m *SingleMarch) Init() {
	m.AddPrepareOpt(func(manager *map_managers.MapManager) {})
	m.AddDoOpt(func(manager *map_managers.MapManager) {})
	m.AddFinishOpt(func(manager *map_managers.MapManager) {})
	m.BaseMarch.Init()
}
