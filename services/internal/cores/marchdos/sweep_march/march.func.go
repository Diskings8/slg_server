package sweep_march

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

// New 创建扫荡行军执行器
//
// 扫荡行军（10003）的生命周期：
//  1. 到达目标 → 验证目标为可采集资源点
//  2. 采集     → TODO: 发放资源收益
//  3. 返回     → 行军自动返回
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		// TODO: 验证目标是否为可采集资源点
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		if m.MarchInfo() == nil {
			return
		}
		// TODO: 采集资源，发放收益
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		// TODO: 推送扫荡结果
	})

	return m
}
