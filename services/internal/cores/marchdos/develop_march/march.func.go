package develop_march

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

// New 创建开发行军执行器
//
// 开发行军（10005）：
//  1. 到达目标 → 对地块执行开发操作
//  2. 结算     → TODO: 改变地块状态
//  3. 返回     → 行军返回
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

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
