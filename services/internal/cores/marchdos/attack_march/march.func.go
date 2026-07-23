package attack_march

import (
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

// New 创建攻击行军执行器
//
// 攻击行军（10001）到达流水线：
//
//	Prepare → 战前合法性校验 → 不合格则直接返回
//	Do      → 战斗结算 → 战损处理 → 占领判定
//	Finish  → 战报推送 → 事件触发
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

	if fromInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetFromMapID()); ok {
		m.SetFromMapInfo(fromInfo)
	}
	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	var battleResult *BattleResult

	// ---- Prepare：战前校验 ----
	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if !checkTargetLegality(mgr, info) {
			info.MarchState = pb_maps_march.MarchState_Back
		}
	})

	// ---- Do：战斗结算 + 战损 + 占领 ----
	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return
		}

		battleResult = settleBattle(mgr, info, info.GetToMapID())
		processBattleResult(mgr, info, battleResult)
	})

	// ---- Finish：战报推送 + 事件触发 ----
	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil || battleResult == nil {
			return
		}

		pushBattleResult(mgr, info, battleResult)
		triggerBattleEvents(mgr, info, battleResult)
	})

	return m
}

// checkTargetLegality 战前目标合法性校验
func checkTargetLegality(mgr *map_managers.MapManager, info *marchs.MarchInfo) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok {
		return false
	}

	if toMapInfo.GetOwnerID() == 0 && toMapInfo.GetOverlayBuilding() == nil {
		return false
	}

	// TODO: 检查目标是否在保护期内
	// TODO: 校验是否是盟友（需接入联盟数据）

	return true
}
