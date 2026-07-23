package attack

import (
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// processBattleResult 处理战斗结果：战损 + 溃败标记 + 占领判定
//
// 在 Do 阶段调用，位于 settleBattle 之后。
func processBattleResult(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil {
		return
	}

	// 1. 应用战损到队伍数据
	applyBattleLosses(result)

	// 2. 更新攻击方行军状态
	updateAttackerState(attacker, result)

	// 3. 处理防御方驻军
	processDefenders(mgr, result)

	// 4. 占领判定
	tryOccupy(mgr, attacker, result)

	// 5. 更新连胜计数
	updateWinStreak(attacker, result)
}

// updateAttackerState 更新攻击方行军状态
func updateAttackerState(attacker *marchs.MarchInfo, result *BattleResult) {
	if result.Attacker == nil {
		return
	}

	if result.Attacker.IsDefeated {
		// 溃败标记：设置行军状态为 Back，队伍带残兵返回
		attacker.MarchState = pb_maps_march.MarchState_Back

		// PVP 溃败：清零 PVP 连胜
		attacker.PVPWinCount = 0
	} else {
		// 胜利且占领 → 转为停留状态
		if result.IsOccupied {
			attacker.MarchState = pb_maps_march.MarchState_Stay
		} else {
			// 胜利但未占领（如仅破墙），仍可继续
			attacker.MarchState = pb_maps_march.MarchState_Stay
		}
	}
}

// processDefenders 处理防御方驻军
func processDefenders(mgr *map_managers.MapManager, result *BattleResult) {
	if result.DefenderMarchUpdates == nil {
		return
	}

	for _, defender := range result.DefenderMarchUpdates {
		if defender == nil {
			continue
		}

		// 驻军被击败：召回
		defender.MarchState = pb_maps_march.MarchState_Back

		// 从目标地块的驻军列表中移除
		toMapID := defender.GetToMapID()
		if toMapID >= 0 {
			attr := mgr.GetMarchManage().MapAttributeGet(toMapID)
			if attr != nil {
				attr.AssistCallBack(defender.GetMarchID())
			}
		}

		// 推送驻军更新
		mgr.UpdateMarchPush(defender)
	}
}

// tryOccupy 尝试占领目标地块
func tryOccupy(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil || !result.IsOccupied || result.Attacker == nil || result.Attacker.IsDefeated {
		return
	}

	toMapID := attacker.GetToMapID()
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return
	}

	// 锁定目标地块进行修改
	if !toMapInfo.TryLock() {
		return
	}
	defer toMapInfo.UnLock()

	// 如果是已有主的地块，先释放
	currentOwner := toMapInfo.GetOwnerID()
	if currentOwner > 0 && currentOwner != attacker.GetFromRoleID() {
		// 从原所有者的 MapAttribute 中移除
		mgr.GetMarchManage().MapAttributeMarchDelete(attacker)
		// 更新行军 ToMapID 为当前地块（不变）
		// 重新绑定到新的所有者
		mgr.GetMarchManage().MapAttributeMarchCreate(attacker)
	}

	// ---- 3. 设置新占领者 ----
	// toMapInfo 已加写锁，直接调用 Occupy 设置
	toMapInfo.Occupy(attacker.GetFromRoleID())
}

// updateWinStreak 更新连胜计数
func updateWinStreak(attacker *marchs.MarchInfo, result *BattleResult) {
	if result.Attacker == nil {
		return
	}

	if result.Attacker.WinCountInc > 0 {
		// PVP 胜利（有驻军防守）
		if result.DefenderMarchUpdates != nil && len(result.DefenderMarchUpdates) > 0 {
			attacker.PVPWinCount++
		} else {
			// PVE 胜利（无驻军，仅建筑/空地）
			attacker.PVEWinCount++
		}
	}
}
