package battle_logics

// 纯逻辑单测 — 8 回合引擎集成（Settle 全链路）。
// 覆盖：PvP 攻胜 / 平局不占领 / 攻城 / 车轮战多防守方 / 车轮战中途停止。

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_cultivate"
)

func testDefender(marchID uint64, slots ...*pb_battle.TeamSlotInfo) *pb_battle.DefenderMarch {
	return &pb_battle.DefenderMarch{MarchId: marchID, Team: &pb_battle.TeamInfo{SlotInfo: slots}}
}

func testDefGroup(marches ...*pb_battle.DefenderMarch) *pb_battle.DefenderGroup {
	return &pb_battle.DefenderGroup{
		GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
		Marches:   marches,
	}
}

// testSlotReloc 带拆迁属性的槽位（攻城用）：reloc=cur_val, camp=加点
func testSlotReloc(slot int32, heroID uint64, alive, mov, atk, def, intel, rng, reloc, camp uint32) *pb_battle.TeamSlotInfo {
	s := testSlot(slot, heroID, alive, mov, atk, def, intel, rng)
	s.HeroInfo.AttrRelocation = &pb_cultivate.Cultivate{CurVal: reloc, AddValCamp: camp}
	return s
}

// TestSettlePvPAttackerWin 攻方大营击败守方大营 → 占领（无建筑 PvE）
func TestSettlePvPAttackerWin(t *testing.T) {
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{SlotInfo: []*pb_battle.TeamSlotInfo{
			testSlot0(1001, 100, 50, 100, 80, 60, 5),
		}},
		DefenderGroups: []*pb_battle.DefenderGroup{
			testDefGroup(testDefender(111, testSlot0(2001, 100, 40, 100, 80, 60, 5))),
		},
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("期望攻击方获胜，实际溃败/停止")
	}
	if !rsp.GetOccupied() {
		t.Fatalf("无建筑 PvE 期望占领")
	}
	if len(rsp.GetDefeatedDefenders()) != 1 || rsp.GetDefeatedDefenders()[0].GetMarchId() != 111 {
		t.Fatalf("期望被击败防守方 march=111，实际 %+v", rsp.GetDefeatedDefenders())
	}
	// 攻方大营承伤（守方大营在守方行动序前已阵亡 → 攻方无损）
	if got := rsp.GetResults().GetResults()[0].GetAttacker().GetTeamInfo().GetSlotInfo()[0].GetCurAliveNum(); got != 100 {
		t.Errorf("攻方大营有效兵力 = %d, want 100", got)
	}
}

// TestSettleDraw 攻方射程不足够不到守方大营 → 平局，不占领
func TestSettleDraw(t *testing.T) {
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{SlotInfo: []*pb_battle.TeamSlotInfo{
			testSlot0(1001, 100, 50, 100, 80, 60, 3),
		}},
		DefenderGroups: []*pb_battle.DefenderGroup{
			testDefGroup(testDefender(222, testSlot0(2001, 100, 40, 100, 80, 60, 3))),
		},
	}

	rsp := Settle(req)

	if rsp.GetAttackerWin() {
		t.Fatalf("平局期望攻方不获胜")
	}
	if rsp.GetOccupied() {
		t.Fatalf("平局期望不占领")
	}
	if len(rsp.GetDefeatedDefenders()) != 0 {
		t.Fatalf("平局期望无被击败防守方")
	}
}

// TestSettleSiegeBroken 攻城：拆迁 150 > 耐久 100 → 占领
func TestSettleSiegeBroken(t *testing.T) {
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{SlotInfo: []*pb_battle.TeamSlotInfo{
			testSlotReloc(0, 1001, 100, 50, 100, 80, 60, 5, 20, 130), // 拆20 + 加点130 = 150
		}},
		HasBuilding: true,
		BuildingHp:  100,
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() {
		t.Fatalf("无防守期望攻方获胜")
	}
	if !rsp.GetOccupied() {
		t.Fatalf("拆迁 150 > 耐久 100，期望占领")
	}
	if rsp.GetBuildingDamage() != 150 {
		t.Fatalf("期望建筑伤害 150，实际 %d", rsp.GetBuildingDamage())
	}
}

// TestSettleWheelBattle 车轮战：攻方依次击败两个守方行军 → 占领
func TestSettleWheelBattle(t *testing.T) {
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{SlotInfo: []*pb_battle.TeamSlotInfo{
			testSlot0(1001, 100, 50, 100, 80, 60, 5),
		}},
		DefenderGroups: []*pb_battle.DefenderGroup{
			testDefGroup(
				testDefender(111, testSlot0(2001, 100, 40, 100, 80, 60, 5)),
				testDefender(222, testSlot0(2002, 100, 40, 100, 80, 60, 5)),
			),
		},
	}

	rsp := Settle(req)

	if !rsp.GetAttackerWin() || !rsp.GetOccupied() {
		t.Fatalf("期望攻方连胜两场并占领，win=%v occ=%v", rsp.GetAttackerWin(), rsp.GetOccupied())
	}
	if len(rsp.GetDefeatedDefenders()) != 2 {
		t.Fatalf("期望 2 个被击败防守方，实际 %d", len(rsp.GetDefeatedDefenders()))
	}
	// 两场战斗 + 最后的 PvE 占领层
	if len(rsp.GetResults().GetResults()) != 3 {
		t.Fatalf("期望 3 个结果（2 场车轮战 + PvE 层），实际 %d", len(rsp.GetResults().GetResults()))
	}
}

// TestSettleWheelBattleStopsOnLoss 车轮战：攻方击败第一场、第二场被守方先手反杀 → 停止（不占领）
func TestSettleWheelBattleStopsOnLoss(t *testing.T) {
	req := &pb_battle.BattleSettleReq{
		AttackerTeam: &pb_battle.TeamInfo{SlotInfo: []*pb_battle.TeamSlotInfo{
			testSlot0(1001, 100, 50, 100, 80, 60, 5),
		}},
		DefenderGroups: []*pb_battle.DefenderGroup{
			testDefGroup(
				testDefender(111, testSlot0(2001, 100, 40, 100, 80, 60, 5)), // 攻方先手可击败
				testDefender(222, testSlot0(2002, 100, 60, 100, 80, 60, 5)), // 守方先手反杀攻方大营
			),
		},
	}

	rsp := Settle(req)

	if rsp.GetAttackerWin() {
		t.Fatalf("攻方大营被反杀，期望不获胜")
	}
	if rsp.GetOccupied() {
		t.Fatalf("攻方未突破全部防守期望不占领")
	}
	// 第一场守方被击败（回城），第二场守方存活
	if len(rsp.GetDefeatedDefenders()) != 1 || rsp.GetDefeatedDefenders()[0].GetMarchId() != 111 {
		t.Fatalf("期望仅 march=111 被击败，实际 %+v", rsp.GetDefeatedDefenders())
	}
}
