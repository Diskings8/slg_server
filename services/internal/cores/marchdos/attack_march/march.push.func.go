package attack_march

import (
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// pushBattleResult 推送战斗结果
//
// 通过现有推送通道通知攻守双方：
//   - UpdateMarchPush：推送更新后的攻击方行军状态
//   - UpdateMapPush：推送目标地块变化
func pushBattleResult(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if attacker == nil {
		return
	}

	fromMapID := attacker.GetFromMapID()
	toMapID := attacker.GetToMapID()

	// 1. 推送更新后的攻击方行军
	mgr.UpdateMarchPush(attacker)

	// 2. 推送目标地块变化
	if toMapID >= 0 {
		mgr.UpdateMapPush(toMapID)
	}

	if fromMapID >= 0 {
		mgr.UpdateMapPush(fromMapID)
	}
}
