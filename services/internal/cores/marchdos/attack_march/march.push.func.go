package attack_march

import (
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// pushBattleResult 推送战斗结果
//
// 通知攻守双方：
//   - 攻击方：行军状态更新 + 战斗结果
//   - 防守方：地块被攻击的预警 / 地块变更
func pushBattleResult(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, result *BattleResult) {
	if attacker == nil {
		return
	}

	// 默认防守方胜。只要有一层攻击方未溃败，判定攻击方胜。
	attackerWin := false
	for _, layer := range result.Layers {
		if layer.Attacker != nil && !layer.Attacker.GetIsDefeated() {
			attackerWin = true
			break
		}
	}

	// 使用 PushBattleResult 统一推送（区分攻守双方）
	mgr.PushBattleResult(attacker, attackerWin)
}
