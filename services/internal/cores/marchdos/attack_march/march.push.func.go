package attack

import (
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// pushBattleResult 推送战斗结果
//
// 通过现有推送通道通知攻守双方：
//   - UpdateMarchPush：推送更新后的攻击方行军状态
//   - UpdateMapPush：推送目标地块变化
//
// TODO: P1-8 增加区分攻防双方的战报推送（PushWarInformation）
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

	// 3. 推送出发地变化
	if fromMapID >= 0 {
		mgr.UpdateMapPush(fromMapID)
	}

	// 4. 推送防御方驻军更新（已在 processDefenders 中逐个推送）
	_ = result
}
