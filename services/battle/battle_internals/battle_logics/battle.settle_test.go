package battle_logics

// 纯逻辑单测 — 校验战斗计算（多轮）行为。
// 覆盖：PvP 攻胜扣损 / 攻败幸存 / 受伤英雄跳过 / 攻城破与不破 / PvE / 攻败不再打建筑 / 多轮轮次 / 防守方伤亡。

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
)

// slot 构造一个队伍槽位
func slot(id int32, alive, max uint32, status pb_hero.Status, relocation uint32) *pb_battle.TeamSlotInfo {
	return &pb_battle.TeamSlotInfo{
		SlotId:        id,
		MaxSoldierNum: max,
		CurAliveNum:   alive,
		CurInjuredNum: 0,
		HeroInfo: &pb_hero.HeroInfo{
			CurStatus:      status,
			AttrRelocation: &pb_cultivate.Cultivate{CurVal: relocation},
		},
	}
}

func team(slots ...*pb_battle.TeamSlotInfo) *pb_battle.TeamInfo {
	return &pb_battle.TeamInfo{SlotInfo: slots}
}

func defender(marchID uint64, t *pb_battle.TeamInfo) *pb_battle.DefenderMarch {
	return &pb_battle.DefenderMarch{MarchId: marchID, Team: t}
}

// totalAlive 统计快照总存活数，便于断言
func totalAlive(slots []*pb_battle.TeamSlotInfo) uint32 {
	var sum uint32
	for _, s := range slots {
		sum += s.GetCurAliveNum()
	}
	return sum
}

// lastAttackerAlive 最后一层结果的攻击方存活数（最终战斗状态）
func lastAttackerAlive(rsp *pb_battle.BattleSettleRsp) uint32 {
	results := rsp.GetResults().GetResults()
	if len(results) == 0 {
		return 0
	}
	last := results[len(results)-1]
	return totalAlive(last.GetAttacker().GetTeamInfo().GetSlotInfo())
}

func TestPvPAttackerWin(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 0), slot(2, 100, 100, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches: []*pb_battle.DefenderMarch{defender(111, team(slot(1, 50, 50, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜，实际溃败")
	}
	if !rsp.GetOccupied() {
		t.Fatalf("无建筑 PvE 期望占领")
	}
	// 多轮(PvP) + PvE：至少 2 条结果
	if len(rsp.GetResults().GetResults()) < 2 {
		t.Fatalf("期望多轮结果 >= 2，实际 %d", len(rsp.GetResults().GetResults()))
	}
	// 攻方 200 vs 守 50：多轮磨平守方到下限，攻方最终存活 150
	after := lastAttackerAlive(rsp)
	if after != 150 {
		t.Fatalf("期望最终存活 150，实际 %d", after)
	}
	// 被击败防守方
	if len(rsp.GetDefeatedDefenders()) != 1 || rsp.GetDefeatedDefenders()[0].GetMarchId() != 111 {
		t.Fatalf("期望被击败防守方 march=111，实际 %+v", rsp.GetDefeatedDefenders())
	}
	// 入参未被污染
	if totalAlive(req.GetAttackerTeam().GetSlotInfo()) != 200 {
		t.Fatalf("入参队伍被污染，期望 200 实际 %d", totalAlive(req.GetAttackerTeam().GetSlotInfo()))
	}
}

func TestPvPAttackerDefeated(t *testing.T) {
	attacker := team(slot(1, 50, 50, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		HasBuilding:  true, // 攻败后不应进入攻城
		BuildingHp:   9999,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_STAY_IDLE,
				Marches: []*pb_battle.DefenderMarch{defender(222, team(slot(1, 200, 200, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	if rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方溃败")
	}
	if rsp.GetOccupied() {
		t.Fatalf("攻败期望不占领")
	}
	if rsp.GetBuildingDamage() != 0 {
		t.Fatalf("攻败后不应打建筑，期望 damage=0 实际 %d", rsp.GetBuildingDamage())
	}
	// 攻方 50 vs 守 200：多轮被磨到下限 1
	if after := lastAttackerAlive(rsp); after != 1 {
		t.Fatalf("期望攻败幸存 1，实际 %d", after)
	}
	if len(rsp.GetResults().GetResults()) != 2 {
		t.Fatalf("攻败期望 2 轮结果（不再打建筑），实际 %d", len(rsp.GetResults().GetResults()))
	}
}

func TestInjuredHeroSkipped(t *testing.T) {
	attacker := team(
		slot(1, 100, 100, pb_hero.Status_Normal, 0),
		slot(2, 100, 100, pb_hero.Status_Injured, 0),
	)
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches: []*pb_battle.DefenderMarch{defender(333, team(slot(1, 50, 50, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜")
	}
	// 最终状态取最后一层结果（多轮末尾 / PvE 层）
	results := rsp.GetResults().GetResults()
	after := results[len(results)-1].GetAttacker().GetTeamInfo().GetSlotInfo()
	// 核心不变量：受伤槽位完全不参与扣损
	if after[1].GetCurAliveNum() != 100 {
		t.Fatalf("slot2(受伤) 期望不被扣损保持 100，实际 %d", after[1].GetCurAliveNum())
	}
	// 正常槽位承伤（具体值受浮点逐槽截断影响，用范围断言）
	if after[0].GetCurAliveNum() == 0 || after[0].GetCurAliveNum() >= 100 {
		t.Fatalf("slot1(正常) 期望承伤后 <100 且 >0，实际 %d", after[0].GetCurAliveNum())
	}
}

func TestSiegeBroken(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 150))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		HasBuilding:  true,
		BuildingHp:   100,
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜")
	}
	if !rsp.GetOccupied() {
		t.Fatalf("拆迁 150 > 耐久 100，期望占领")
	}
	if rsp.GetBuildingDamage() != 150 {
		t.Fatalf("期望建筑伤害 150，实际 %d", rsp.GetBuildingDamage())
	}
}

func TestSiegeNotBroken(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 150))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		HasBuilding:  true,
		BuildingHp:   200,
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜")
	}
	if rsp.GetOccupied() {
		t.Fatalf("拆迁 150 <= 耐久 200，期望不占领")
	}
	if rsp.GetBuildingDamage() != 150 {
		t.Fatalf("期望建筑伤害 150（未击破也造成伤害），实际 %d", rsp.GetBuildingDamage())
	}
}

func TestPvEOccupied(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{AttackerTeam: attacker}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() || !rsp.GetOccupied() {
		t.Fatalf("PvE 无防守无建筑期望直接占领")
	}
	if rsp.GetBuildingDamage() != 0 {
		t.Fatalf("PvE 期望无建筑伤害")
	}
}

func TestNoAliveAttackerDefeatedImmediately(t *testing.T) {
	attacker := team(slot(1, 0, 100, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		HasBuilding:  true,
		BuildingHp:   10,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches: []*pb_battle.DefenderMarch{defender(444, team(slot(1, 100, 100, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	if rsp.GetAttackerWin() {
		t.Fatalf("攻击方无存活士兵期望直接溃败")
	}
	if len(rsp.GetResults().GetResults()) != 0 {
		t.Fatalf("无战力不产出层结果，期望 0 层，实际 %d", len(rsp.GetResults().GetResults()))
	}
	if len(rsp.GetDefeatedDefenders()) != 0 {
		t.Fatalf("未交战期望无被击败防守方")
	}
}

// TestMultiRoundRounds 多轮：PvP 应产生多轮结果（>1），每轮攻守双方都记录战损
func TestMultiRoundRounds(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 0), slot(2, 100, 100, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches: []*pb_battle.DefenderMarch{defender(111, team(slot(1, 50, 50, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	rounds := rsp.GetResults().GetResults()
	// PvP 部分至少 2 轮（200 vs 50 需多轮磨平）
	pvpRounds := len(rounds) - 1 // 减去最后的 PvE 层
	if pvpRounds < 2 {
		t.Fatalf("期望 PvP 至少 2 轮，实际 %d", pvpRounds)
	}
	// 每轮攻方有战损记录（KilledSoldiers > 0）
	first := rounds[0].GetAttacker()
	if first.GetKilledSoldiers() == 0 {
		t.Fatalf("第 1 轮攻方期望有战损，实际 %d", first.GetKilledSoldiers())
	}
}

// TestDefenderCasualties 多轮：防守方承伤后 TeamAfter 应包含战后（减少）的队伍
func TestDefenderCasualties(t *testing.T) {
	attacker := team(slot(1, 100, 100, pb_hero.Status_Normal, 0))
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: attacker,
		DefenderGroups: []*pb_battle.DefenderGroup{
			{GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches: []*pb_battle.DefenderMarch{defender(555, team(slot(1, 50, 50, pb_hero.Status_Normal, 0)))}},
		},
	}

	rsp := Settle(req)

	if len(rsp.GetDefeatedDefenders()) != 1 {
		t.Fatalf("期望 1 个被击败防守方")
	}
	dd := rsp.GetDefeatedDefenders()[0]
	if dd.GetTeamAfter() == nil {
		t.Fatalf("防守方战后队伍快照 TeamAfter 不应为空（多轮防守方承伤）")
	}
	alive := totalAlive(dd.GetTeamAfter().GetSlotInfo())
	if alive >= 50 {
		t.Fatalf("防守方应承伤（50 → <50），实际 %d", alive)
	}
}
