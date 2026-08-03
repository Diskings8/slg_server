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
//
// attackerWin 由 battle 节点结算结果给出（battleRsp.AttackerWin）。
func pushBattleResult(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, attackerWin bool) {
	if attacker == nil {
		return
	}

	// 使用 PushBattleResult 统一推送（区分攻守双方）
	mgr.PushBattleResult(attacker, attackerWin)
}
