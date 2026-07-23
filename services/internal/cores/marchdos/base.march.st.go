package marchdos

import (
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// BaseMarch 行军执行器基类，封装行军锁定和解锁逻辑，提供来源和目标地图格子的并发安全访问
type BaseMarch struct {
	marchManage   *marchs.MarchInfoManager // 行军管理
	mgr           *map_managers.MapManager // MarchDoFuncHandleI.Do() 使用的管理器
	fromMapInfo   *map_datas.MapInfo       // 来源地图信息
	toMapInfo     *map_datas.MapInfo       // 目标地图信息
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

// SetManager 设置地图管理器，供 Do() error 使用
func (m *BaseMarch) SetManager(mm *map_managers.MapManager) {
	m.mgr = mm
}

// Do 执行行军流程（MarchDoFuncHandleI 接口实现）
//
// 按顺序执行 prepareOpts → doOpts → finishOpts。
// 需要先通过 SetManager 设置地图管理器，否则 panic。
func (m *BaseMarch) Do() error {
	if !m.hadInit {
		panic("marchdos: Do called before Init, missing BaseMarch.Init()?")
	}
	if m.mgr == nil {
		panic("marchdos: Do called before SetManager")
	}
	for _, v := range m.prepareOpts {
		v(m.mgr)
	}
	for _, v := range m.doOpts {
		v(m.mgr)
	}
	for _, v := range m.finishOpts {
		v(m.mgr)
	}
	return nil
}

// DoWithManager 手动指定管理器执行行军流程（兼容内部调用）
func (m *BaseMarch) DoWithManager(mapManager *map_managers.MapManager) {
	if !m.hadInit {
		panic("marchdos: DoWithManager called before Init")
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
