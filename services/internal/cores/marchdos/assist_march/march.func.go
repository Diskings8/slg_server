package assist

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

func init() {
	marchdos.RegisterMarchFactory(cores_declarations.MarchTypeAssist, New)
}

// New 创建驻守行军执行器
//
// 驻守行军（10002）的生命周期：
//  1. 到达目标 → 在目标地块注册驻军
//  2. 防守     → 目标被攻击时参与防守
//  3. 返回     → 召回或主动撤离
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		// TODO: 检查目标是否为盟友地块、驻军上限等
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		if m.MarchInfo() == nil {
			return
		}
		// 在目标地块注册驻军
		mgr.GetMarchManage().MapAttributeMarchCreate(m.MarchInfo())
		// TODO: 通知被驻守方
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info != nil {
			mgr.UpdateMapPush(info.GetFromMapID(), info.GetToMapID())
		}
	})

	return m
}
