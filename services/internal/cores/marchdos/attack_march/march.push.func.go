package attack_march

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// buildMarchBattleResult 汇总 battle 节点战果：attacker_win + 每英雄总经验（多轮累计）。
// 由攻击行军 Finish 存入 MarchInfo.BattleResult，到达事件发布时随 MarchEvent 回传 game。
func buildMarchBattleResult(rsp *pb_battle.BattleSettleRsp) *pb_redis_stream.MarchBattleResult {
	if rsp == nil {
		return nil
	}
	result := &pb_redis_stream.MarchBattleResult{
		AttackerWin: rsp.GetAttackerWin(),
	}
	expMap := make(map[uint64]uint32)
	for _, one := range rsp.GetResults().GetResults() {
		for _, he := range one.GetHeroExp() {
			expMap[he.GetHeroId()] += he.GetExp()
		}
	}
	for heroID, exp := range expMap {
		result.HeroExp = append(result.HeroExp, &pb_redis_stream.HeroExpItem{
			HeroId: heroID,
			Exp:    exp,
		})
	}
	return result
}

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
