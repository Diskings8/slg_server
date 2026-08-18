package battle_logics

// 分层战斗 — 8 回合回合制引擎（battle.framework.func.go）的调度层。
// 车轮战：攻方依次与各防守行军各打一场，战损累积，全胜后进入攻城/PvE。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
)

// fightOneMarch 攻方 vs 单个守方行军的一场 8 回合战斗。
//
// 攻方队伍快照被原地更新（战损累积），供下一场车轮战使用；防守方克隆后结算。
// 返回 nil 表示守方无队伍（无交战）。
func fightOneMarch(attackerSlots []*pb_battle.TeamSlotInfo, defender *pb_battle.DefenderMarch) *pb_battle.OneBattleResult {
	if defender == nil || len(defender.GetTeam().GetSlotInfo()) == 0 {
		return nil
	}
	return runBattle(attackerSlots, cloneSlots(defender.GetTeam().GetSlotInfo()))
}

// resolveDefendersLayers 车轮战：攻方依次与各防守行军各打一场。
//
//   - 攻方赢一场 → 防守方被击败（TeamAfter 写回伤亡），继续下一场
//   - 攻方大营败 → 攻方溃败，停止后续
//   - 平局（8回合双方大营均有兵）→ 攻方未能突破，停止（不溃败但也不占领）
//   - 同归于尽 → 攻方溃败
//
// 返回被击败的防守方，以及攻方是否已停止（未突破全部防守）。
func resolveDefendersLayers(req *pb_battle.BattleSettleReq, attackerSlots []*pb_battle.TeamSlotInfo,
	result *pb_battle.BattleResults) (defeated []*pb_battle.DefeatedDefender, attackerStopped bool) {

	for _, group := range req.GetDefenderGroups() {
		if group == nil || len(group.GetMarches()) == 0 {
			continue
		}
		for _, d := range group.GetMarches() {
			if d == nil {
				continue
			}
			one := fightOneMarch(attackerSlots, d)
			if one == nil {
				continue
			}
			result.Results = append(result.Results, one)

			// 本场经验：敌方平均等级 × 击杀(死亡兵) × 系数 ÷ 参战英雄
			grantBattleExp(attackerSlots, one.GetDefender().GetTeamInfo().GetSlotInfo(),
				one.GetAttacker().GetKilledSoldiers(), one)

			attDead := baseDeadAfter(one.GetAttacker())
			defDead := baseDeadAfter(one.GetDefender())

			if defDead {
				// 防守方大营败 → 被击败（回城）
				defeated = append(defeated, &pb_battle.DefeatedDefender{
					MarchId:   d.GetMarchId(),
					TeamAfter: one.GetDefender().GetTeamInfo(),
				})
				if attDead {
					return defeated, true // 同归于尽
				}
				continue // 攻方获胜，进入下一场
			}
			// 防守方未败：攻方败或平局 → 攻方停止
			return defeated, true
		}
	}
	return defeated, false
}

// baseDeadAfter 战后队伍快照中攻/守方大营(slot1)有效兵力是否归零
func baseDeadAfter(side *pb_battle.BattleSide) bool {
	if side == nil || side.GetTeamInfo() == nil {
		return true
	}
	for _, s := range side.GetTeamInfo().GetSlotInfo() {
		if s.GetSlotId() == 1 { // 大营
			return slotAliveNum(s) == 0
		}
	}
	return true // 无大营视为败
}
