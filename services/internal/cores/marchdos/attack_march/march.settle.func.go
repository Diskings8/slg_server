package attack_march

// 战斗结果应用 — 迁移自 processBattleResult，改为消费 battle 节点返回的 BattleSettleRsp。
// 战斗计算在 battle 节点完成，这里负责把结果写回 worldmap 侧运行时（伤亡/状态/占城/建筑耐久）。

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/loggers"
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

	loggers.Logger.Info("applySettle step1 start", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
	applyAttackerCasualties(attacker, rsp)  // 1. 攻击方伤亡写回
	loggers.Logger.Info("applySettle step2", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
	updateAttackerState(attacker, rsp)      // 2. 状态：Stay / Back
	loggers.Logger.Info("applySettle step3", zap.Uint64("march_id", attacker.GetMarchID().Uint64()), zap.Int32("defenders", int32(len(rsp.GetDefeatedDefenders())))) // TODO debug remove
	processDefeatedDefenders(mgr, rsp)      // 3. 防守方：回城 + AssistCallBack + push
	loggers.Logger.Info("applySettle step4", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
	applyBuildingDamage(mgr, attacker, rsp) // 4. 攻城耐久回写
	loggers.Logger.Info("applySettle step5", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
	tryOccupy(mgr, attacker, rsp)           // 5. 占领
	loggers.Logger.Info("applySettle step6", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
	updateWinStreak(attacker, rsp)          // 6. 连胜（占位）
	loggers.Logger.Info("applySettle done", zap.Uint64("march_id", attacker.GetMarchID().Uint64())) // TODO debug remove
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

	// 归属在抢写锁前读：GetOwnerID 走 RLock，不能与下方 TryLock 的写锁同 goroutine 重入
	// （sync.RWMutex 不可重入，持写锁再 RLock 会自锁死锁，Do 内的 Getter 同理）。
	currentOwner := toMapInfo.GetOwnerID()

	if !toMapInfo.TryLock() {
		return
	}
	defer toMapInfo.UnLock()

	if currentOwner > 0 && currentOwner != attacker.GetFromRoleID() {
		mgr.GetMarchManage().MapAttributeMarchDelete(attacker)
		mgr.GetMarchManage().MapAttributeMarchCreate(attacker)
		// 记录原归属者（瞬态），到达事件回传 game 清理原主资源地快照
		attacker.PrevOwnerID = currentOwner
	}

	toMapInfo.Occupy(attacker.GetFromRoleID())

	// 持久化归属变更（当前持锁，Save 只标记脏，SaveDo 稍后刷盘）
	mgr.GetMapDataManager().Save(toMapInfo)
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
