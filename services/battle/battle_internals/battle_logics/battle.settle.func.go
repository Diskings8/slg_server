package battle_logics

// 战斗结算主入口 — 迁移自 cores/marchdos/attack_march/march.battle.func.go 的
// settleBattle / resolvePvE / buildSideFromTeam，行为保真：
//   - PvP 逐层（assist → stay/idle），攻方溃败则不再结算建筑/PvE
//   - 攻城：broken = 拆迁值 > 建筑耐久（严格大于，对齐 ReduceBuildingsHp）
//   - PvE：无建筑直接占领

import (
	"server.slg.com/api/protocol/pb/pb_battle"
)

// Settle 战斗结算主入口（battle 节点 BattleSettle RPC 的纯逻辑层）。
func Settle(req *pb_battle.BattleSettleReq) *pb_battle.BattleSettleRsp {
	attackerSlots := cloneSlots(req.GetAttackerTeam().GetSlotInfo())
	results := &pb_battle.BattleResults{}

	// 1. PvP 分层（assist → stay/idle）
	defeated, attackerDefeated := resolveDefendersLayers(req, attackerSlots, results)

	// 2. 攻城 / PvE（攻方未被击败才进入）
	var occupied bool
	var buildingDamage uint64
	if !attackerDefeated {
		layer, occ, dmg := settleTarget(req, attackerSlots)
		results.Results = append(results.Results, layer)
		occupied = occ
		buildingDamage = dmg
	}

	results.ResultCount = int32(len(results.Results))

	return &pb_battle.BattleSettleRsp{
		Results:           results,
		AttackerWin:       !attackerDefeated,
		Occupied:          occupied,
		DefeatedDefenders: defeated,
		BuildingDamage:    buildingDamage,
	}
}

// settleTarget 有建筑→攻城，无建筑→PvE。
func settleTarget(req *pb_battle.BattleSettleReq, attackerSlots []*pb_battle.TeamSlotInfo) (
	layer *pb_battle.OneBattleResult, occupied bool, buildingDamage uint64) {

	if req.GetHasBuilding() {
		dmg := relocationVal(attackerSlots)
		broken := dmg > req.GetBuildingHp()
		layer = &pb_battle.OneBattleResult{
			Attacker:   buildSide(attackerSlots),
			Defender:   &pb_battle.BattleSide{},
			IsOccupied: broken,
		}
		return layer, broken, dmg
	}

	// PvE：无建筑 → 占领
	layer = &pb_battle.OneBattleResult{
		Attacker:   buildSide(attackerSlots),
		Defender:   &pb_battle.BattleSide{},
		IsOccupied: true,
	}
	return layer, true, 0
}

// buildSide 由队伍快照构造 BattleSide（对齐 buildSideFromTeam）。
func buildSide(slots []*pb_battle.TeamSlotInfo) *pb_battle.BattleSide {
	return &pb_battle.BattleSide{
		TeamInfo: &pb_battle.TeamInfo{SlotInfo: cloneSlots(slots)},
	}
}
