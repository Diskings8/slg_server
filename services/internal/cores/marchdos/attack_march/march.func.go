package attack

import (
	"server.slg.com/api/protocol/pb/pb_maps_march"
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
// 攻击行军（10001）到达流水线：
//
//	Prepare → 战前合法性校验
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

	// BattleResult 在 Do 中生成，在 Finish 中消费
	var battleResult *BattleResult

	// ---- Prepare：战前校验 ----
	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if !checkTargetLegality(mgr, info, info.GetToMapID()) {
			// 校验不通过，行军返回
			info.MarchState = pb_maps_march.MarchState_Back
		}
	})

	// ---- Do：战斗结算 + 战损 + 占领 ----
	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}

		// 执行战斗结算
		battleResult = settleBattle(mgr, info, info.GetToMapID().Int32())

		// 处理战斗结果（战损、溃败、占领）
		processBattleResult(mgr, info, battleResult)
	})

	// ---- Finish：战报推送 + 事件触发 ----
	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}

		// 推送战报和地图更新
		pushBattleResult(mgr, info, battleResult)

		// 触发战斗事件（预留）
		triggerBattleEvents(mgr, info, battleResult)
	})

	return m
}
