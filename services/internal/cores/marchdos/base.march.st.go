package marchdos

import (
	"server.slg.com/services/internal/cores/map_datas/map_infos"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// BaseMarch 行军执行器基类，封装行军锁定和解锁逻辑，提供来源和目标地图格子的并发安全访问
type BaseMarch struct {
	marchManage   *marchs.MarchInfoManager // 行军管理
	fromMapInfo   *map_infos.MapInfo       // 来源地图信息
	toMapInfo     *map_infos.MapInfo       // 目标地图信息
	marchLockOk   bool
	fromMapLockOk bool
	toMapLockOk   bool
	hadInit       bool
	err           error
	prepareOpts   []func(*map_managers.MapManager) // 先进先出
	doOpts        []func(*map_managers.MapManager) // 先进先出
	finishOpts    []func(*map_managers.MapManager) // 先进先出
}

func (m *BaseMarch) Init() {
	m.AddPrepareOpt(func(manager *map_managers.MapManager) {})
	m.AddDoOpt(func(manager *map_managers.MapManager) {})
	m.AddFinishOpt(func(manager *map_managers.MapManager) {})
	m.hadInit = true
}

func (m *BaseMarch) Do(mapManager *map_managers.MapManager) {
	if !m.hadInit {
		panic("marchdos: Do called before Init, missing BaseMarch.Init()?")
	}
	for _, v := range m.prepareOpts {
		v(mapManager)
	}
	for _, v := range m.doOpts {
		v(mapManager)
	}
	for _, v := range m.finishOpts {
		v(mapManager)
	}
}

func (m *BaseMarch) AddPrepareOpt(f func(*map_managers.MapManager)) {
	m.prepareOpts = append(m.prepareOpts, f)
}

func (m *BaseMarch) AddDoOpt(f func(*map_managers.MapManager)) {
	m.doOpts = append(m.doOpts, f)
}

func (m *BaseMarch) AddFinishOpt(f func(*map_managers.MapManager)) {
	m.finishOpts = append(m.finishOpts, f)
}
