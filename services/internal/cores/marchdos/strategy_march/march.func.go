package strategy

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

func init() {
	marchdos.RegisterMarchFactory(cores_declarations.MarchTypeStrategy, New)
	marchdos.RegisterMarchFactory(cores_declarations.MarchTypeDevelop, newDevelop)
}

// New 创建计略行军执行器
//
// 计略行军（10004）的生命周期：
//  1. 到达目标 → 执行计略效果（侦查、扰乱等）
//  2. 结算     → TODO: 应用计略效果
//  3. 返回     → 行军返回
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch()
	m.SetMarchInfo(marchInfo)
	m.SetMarchManage(mm.GetMarchManage())
	m.SetManager(mm)

	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		// TODO: 检查计略目标合法性
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		if m.MarchInfo() == nil {
			return
		}
		// TODO: 执行计略效果
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		// TODO: 推送计略结果
	})

	return m
}

// newDevelop 创建开发行军执行器
//
// 开发行军（10005）：
//  1. 到达目标 → 对地块执行开发操作
//  2. 结算     → TODO: 改变地块状态
//  3. 返回     → 行军返回
func newDevelop(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch()
	m.SetMarchInfo(marchInfo)
	m.SetMarchManage(mm.GetMarchManage())
	m.SetManager(mm)

	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		// TODO: 检查地块是否可开发
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		if m.MarchInfo() == nil {
			return
		}
		// TODO: 执行开发，更新地块状态
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info != nil {
			mgr.UpdateMapPush(info.GetToMapID())
		}
	})

	return m
}
