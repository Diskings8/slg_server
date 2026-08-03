package attack_march

// 战斗结果应用 — 迁移自 processBattleResult，改为消费 battle 节点返回的 BattleSettleRsp。
// 战斗计算在 battle 节点完成，这里负责把结果写回 worldmap 侧运行时（伤亡/状态/占城/建筑耐久）。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// applyBattleSettleRsp 应用 battle 节点返回的结算结果
func applyBattleSettleRsp(mgr *map_managers.MapManager, attacker *marchs.MarchInfo,
	rsp *pb_battle.BattleSettleRsp) {
	if rsp == nil || attacker == nil {
		return
	}

	applyAttackerCasualties(attacker, rsp)  // 1. 攻击方伤亡写回
	updateAttackerState(attacker, rsp)      // 2. 状态：Stay / Back
	processDefeatedDefenders(mgr, rsp)      // 3. 防守方：回城 + AssistCallBack + push
	applyBuildingDamage(mgr, attacker, rsp) // 4. 攻城耐久回写
	tryOccupy(mgr, attacker, rsp)           // 5. 占领
	updateWinStreak(attacker, rsp)          // 6. 连胜（占位）
}

// applyAttackerCasualties 用最后一层攻击方队伍快照覆盖 MarchInfo.Team
func applyAttackerCasualties(attacker *marchs.MarchInfo, rsp *pb_battle.BattleSettleRsp) {
	last := lastResult(rsp.GetResults())
	if last == nil || last.GetAttacker() == nil {
		return
	}
	if team := last.GetAttacker().GetTeamInfo(); team != nil {
		if attacker.Team == nil {
			attacker.Team = &marchs.Team{}
		}
		attacker.Team.ApplyTeamInfo(team)
	}
}

// lastResult 返回最后一层结果
func lastResult(results *pb_battle.BattleResults) *pb_battle.OneBattleResult {
	if results == nil {
		return nil
	}
	layers := results.GetResults()
	if len(layers) == 0 {
		return nil
	}
	return layers[len(layers)-1]
}

// updateAttackerState 更新攻击方行军状态
//
// 战败时不设 MarchState=Back：召回（交换方向 + 返回 TransitMapID + 重新入队）由
// attack_march Do opt 的 DefeatRecall 统一处理，避免与 callbackSwapDirection 的状态守卫冲突。
func updateAttackerState(attacker *marchs.MarchInfo, rsp *pb_battle.BattleSettleRsp) {
	if rsp == nil || len(rsp.GetResults().GetResults()) == 0 {
		return // 无层结果保持原状态（对齐原实现早退）
	}
	if rsp.GetAttackerWin() {
		attacker.MarchState = pb_maps_march.MarchState_Stay
	} else {
		attacker.PVPWinCount = 0
	}
}

// processDefeatedDefenders 处理被击败的防守方行军
func processDefeatedDefenders(mgr *map_managers.MapManager, rsp *pb_battle.BattleSettleRsp) {
	for _, d := range rsp.GetDefeatedDefenders() {
		if d == nil {
			continue
		}
		defender := mgr.GetMarchManage().GetMarchInfo(cores_declarations.MarchID(d.GetMarchId()))
		if defender == nil {
			continue
		}

		if d.GetTeamAfter() != nil {
			if defender.Team == nil {
				defender.Team = &marchs.Team{}
			}
			defender.Team.ApplyTeamInfo(d.GetTeamAfter())
		}

		defender.MarchState = pb_maps_march.MarchState_Back

		toMapID := defender.GetToMapID()
		if toMapID >= 0 {
			attr := mgr.GetMarchManage().MapAttributeGet(toMapID)
			if attr != nil {
				attr.AssistCallBack(defender.GetMarchID())
			}
		}
		mgr.UpdateMarchPush(defender)
	}
}

// applyBuildingDamage 攻城耐久回写。
// battle 节点只返回伤害值，worldmap 侧建筑内存需就地扣血（原 BeAttack 就地扣血行为）。
func applyBuildingDamage(mgr *map_managers.MapManager, attacker *marchs.MarchInfo,
	rsp *pb_battle.BattleSettleRsp) {
	if rsp.GetBuildingDamage() == 0 {
		return
	}
	toMapID := attacker.GetToMapID()
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return
	}
	overlay := toMapInfo.GetOverlayBuilding()
	if overlay == nil {
		return
	}
	if building := overlay.GetBuilding(); building != nil {
		building.ReduceBuildingsHp(rsp.GetBuildingDamage())
	}
}

// tryOccupy 尝试占领目标地块
func tryOccupy(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, rsp *pb_battle.BattleSettleRsp) {
	if !rsp.GetOccupied() || !rsp.GetAttackerWin() {
		return
	}

	toMapID := attacker.GetToMapID()
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return
	}

	if !toMapInfo.TryLock() {
		return
	}
	defer toMapInfo.UnLock()

	currentOwner := toMapInfo.GetOwnerID()
	if currentOwner > 0 && currentOwner != attacker.GetFromRoleID() {
		mgr.GetMarchManage().MapAttributeMarchDelete(attacker)
		mgr.GetMarchManage().MapAttributeMarchCreate(attacker)
	}

	toMapInfo.Occupy(attacker.GetFromRoleID())
}

// updateWinStreak 更新连胜计数（占位：win_count_inc 恒 0，保行为）
func updateWinStreak(attacker *marchs.MarchInfo, rsp *pb_battle.BattleSettleRsp) {
	if rsp.GetResults().GetWinCountInc() == 0 {
		return
	}
	if len(rsp.GetDefeatedDefenders()) > 0 {
		attacker.PVPWinCount++
	} else {
		attacker.PVEWinCount++
	}
}
