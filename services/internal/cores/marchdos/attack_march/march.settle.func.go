package attack_march

import (
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// processBattleResult 处理战斗结果
func processBattleResult(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil {
		return
	}

	// 1. 更新攻击方行军状态
	updateAttackerState(attacker, result)

	// 2. 处理防御方行军
	processDefeatedMarches(mgr, result)

	// 3. 占领判定（最后一层决定）
	tryOccupy(mgr, attacker, result)

	// 4. 更新连胜计数
	updateWinStreak(attacker, result)
}

// updateAttackerState 更新攻击方行军状态
func updateAttackerState(attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil || len(result.Layers) == 0 {
		return
	}

	// 取最后一层的结果判定是否溃败
	lastLayer := result.Layers[len(result.Layers)-1]
	if lastLayer.Attacker != nil && lastLayer.Attacker.IsDefeated {
		attacker.MarchState = pb_maps_march.MarchState_Back
		attacker.PVPWinCount = 0
	} else {
		attacker.MarchState = pb_maps_march.MarchState_Stay
	}
}

// processDefeatedMarches 处理被击败的防御方行军
func processDefeatedMarches(mgr *map_managers.MapManager, result *BattleResult) {
	for _, layer := range result.Layers {
		for _, defender := range layer.DefeatedMarches {
			if defender == nil {
				continue
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
}

// tryOccupy 尝试占领目标地块
func tryOccupy(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil || len(result.Layers) == 0 {
		return
	}

	// 最后一层决定占领
	lastLayer := result.Layers[len(result.Layers)-1]
	if !lastLayer.IsOccupied {
		return
	}
	if lastLayer.Attacker != nil && lastLayer.Attacker.IsDefeated {
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

// updateWinStreak 更新连胜计数
func updateWinStreak(attacker *marchs.MarchInfo, result *BattleResult) {
	if result == nil || result.WinCountInc == 0 {
		return
	}

	hasDefender := false
	for _, layer := range result.Layers {
		if len(layer.DefeatedMarches) > 0 {
			hasDefender = true
			break
		}
	}

	if hasDefender {
		attacker.PVPWinCount++
	} else {
		attacker.PVEWinCount++
	}
}
