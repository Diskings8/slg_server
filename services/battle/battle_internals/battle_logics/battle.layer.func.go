package battle_logics

// 分层战斗 — 迁移自 cores/marchdos/attack_march/march.battle.func.go 的
// fightLayer / resolveDefendersLayer，操作对象改为 pb_battle 快照。
// 战斗升级为多轮：攻守双方每轮互相承伤，直到一方全灭 / 僵持 / 轮次上限。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
)

// maxBattleRounds 单层战斗轮次上限
const maxBattleRounds = 50

// fightLayer 攻击方与某一层防守方多轮交战。
//
// 每轮攻守双方按对方战力占比互相造成伤亡，直到一方全灭 / 僵持无进展 / 达到轮次上限。
// 每轮产出一条 OneBattleResult 追加到 result.Results。
//
// 返回：
//   - attackerDefeated: 攻击方是否溃败（true 时后续层/攻城不再结算）
//   - defenderTeams: 各防守方战后队伍快照（与 defenders 中的非 nil 项一一对应；nil 表示该层未交战）
func fightLayer(attackerSlots []*pb_battle.TeamSlotInfo, defenders []*pb_battle.DefenderMarch,
	result *pb_battle.BattleResults) (attackerDefeated bool, defenderTeams [][]*pb_battle.TeamSlotInfo) {

	defTeams := make([][]*pb_battle.TeamSlotInfo, 0, len(defenders))
	for _, d := range defenders {
		if d == nil {
			continue
		}
		defTeams = append(defTeams, cloneSlots(d.GetTeam().GetSlotInfo()))
	}
	if len(defTeams) == 0 {
		return false, nil
	}

	attPower := aliveSoldierCount(attackerSlots)
	if attPower == 0 {
		return true, nil
	}
	if aliveSoldierCountTeams(defTeams) == 0 {
		return false, nil
	}

	for round := 1; round <= maxBattleRounds; round++ {
		attPower := aliveSoldierCount(attackerSlots)
		defPower := aliveSoldierCountTeams(defTeams)
		if attPower == 0 {
			return true, defTeams
		}
		if defPower == 0 {
			return false, defTeams
		}

		// 本轮互损：按对方战力占比
		attLoss := uint64(float64(attPower) * (float64(defPower) / float64(attPower+defPower)))
		defLoss := uint64(float64(defPower) * (float64(attPower) / float64(attPower+defPower)))
		if attLoss > attPower {
			attLoss = attPower
		}
		if defLoss > defPower {
			defLoss = defPower
		}

		if attLoss == 0 && defLoss == 0 {
			break // 无进展（士兵已减到下限 / 战力悬殊到无法造成伤害）
		}

		applyLossesToSlots(attackerSlots, attPower, attPower-attLoss)
		applyDefenderLosses(defTeams, defPower, defPower-defLoss)

		result.Results = append(result.Results, &pb_battle.OneBattleResult{
			Attacker: &pb_battle.BattleSide{
				KilledSoldiers: defLoss,
				TeamInfo:       &pb_battle.TeamInfo{SlotInfo: cloneSlots(attackerSlots)},
			},
			Defender: &pb_battle.BattleSide{
				KilledSoldiers: attLoss,
				TeamInfo:       &pb_battle.TeamInfo{SlotInfo: cloneSlotsTeams(defTeams)},
			},
		})
	}

	// 轮次上限 / 僵持：剩余战力高者胜，平局攻方胜（对齐单轮 attPower >= defPower 攻胜）
	attPower = aliveSoldierCount(attackerSlots)
	defPower := aliveSoldierCountTeams(defTeams)
	return attPower < defPower, defTeams
}

// resolveDefendersLayers 逐层攻克防守方（assist → stay/idle），每层多轮。
//
// 返回被击败的防守方（实际交战层全部防守方，对齐原 DefeatedMarches 语义，
// 含战后队伍快照供 cores 写回伤亡），以及攻击方是否已溃败（溃败则后续层不再结算）。
func resolveDefendersLayers(req *pb_battle.BattleSettleReq, attackerSlots []*pb_battle.TeamSlotInfo,
	result *pb_battle.BattleResults) (defeated []*pb_battle.DefeatedDefender, attackerDefeated bool) {

	for _, group := range req.GetDefenderGroups() {
		if group == nil || len(group.GetMarches()) == 0 {
			continue
		}

		layerDefeated, defTeams := fightLayer(attackerSlots, group.GetMarches(), result)
		if defTeams != nil {
			// 该层实际交战：全部防守方按战后队伍快照召回
			j := 0
			for _, d := range group.GetMarches() {
				if d == nil {
					continue
				}
				dd := &pb_battle.DefeatedDefender{MarchId: d.GetMarchId()}
				if j < len(defTeams) && defTeams[j] != nil {
					dd.TeamAfter = &pb_battle.TeamInfo{SlotInfo: defTeams[j]}
				}
				defeated = append(defeated, dd)
				j++
			}
		}
		if layerDefeated {
			// 攻方已败，后续层不再结算
			return defeated, true
		}
	}
	return defeated, false
}
