package attack

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

func init() {
	marchdos.RegisterMarchFactory(cores_declarations.MarchTypeAttack, New)
}

// New 创建攻击行军执行器
//
// 攻击行军（10001）的生命周期：
//  1. 到达目标地块 → 验证目标合法性
//  2. 执行战斗     → TODO: 接入战斗服务
//  3. 处理结果     → 清理地块、推送更新
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch()
	m.SetMarchInfo(marchInfo)
	m.SetMarchManage(mm.GetMarchManage())
	m.SetManager(mm)

	if fromInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetFromMapID()); ok {
		m.SetFromMapInfo(fromInfo)
	}
	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		// TODO: 检查目标是否可攻击（保护期、合法性等）
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		// TODO: 接入战斗服务，结算战斗
		// 1. 构建攻守双方数据
		// 2. 调用战斗 gRPC 服务
		// 3. 处理结果：扣血、释放地块、掉落等
		if m.MarchInfo() == nil {
			return
		}
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info != nil {
			mgr.UpdateMapPush(info.GetFromMapID(), info.GetToMapID())
		}
	})

	return m
}
