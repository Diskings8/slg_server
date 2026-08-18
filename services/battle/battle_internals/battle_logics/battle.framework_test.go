package battle_logics

// 8 回合战斗引擎单测 — 大营胜负 / 伤兵 / 同归于尽 / 平局 / 追击技能 / 攻击距离。

import (
	"os"
	"testing"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_skill"
)

func TestMain(m *testing.M) {
	_ = game_conf.InitDefault()
	os.Exit(m.Run())
}

// testSlot 构造战斗槽位（槽位1=大营；属性/实例ID/兵力/攻击距离由 HeroInfo 快照携带）
func testSlot(slot int32, heroID uint64, alive uint32, mov, atk, def, intel, rng uint32, skills ...*pb_skill.Skill) *pb_battle.TeamSlotInfo {
	return &pb_battle.TeamSlotInfo{
		SlotId: slot,
		HeroInfo: &pb_hero.HeroInfo{
			ConfigId:         1,
			CurLevel:         1,
			CurStatus:        pb_hero.Status_Normal,
			HeroId:           heroID,
			AttackRange:      rng,
			AttrAttack:       &pb_cultivate.Cultivate{CurVal: atk},
			AttrDefense:      &pb_cultivate.Cultivate{CurVal: def},
			AttrIntelligence: &pb_cultivate.Cultivate{CurVal: intel},
			AttrMovement:     &pb_cultivate.Cultivate{CurVal: mov},
			Skills:           skills,
			SoldierInfo: &pb_hero.SoldierInfo{
				MaxSoldierNum: alive,
				CurAliveNum:   alive,
				CurInjuredNum: 0,
			},
		},
	}
}

func testSlot0(heroID uint64, alive uint32, mov, atk, def, intel, rng uint32, skills ...*pb_skill.Skill) *pb_battle.TeamSlotInfo {
	return testSlot(1, heroID, alive, mov, atk, def, intel, rng, skills...) // 大营 = slot 1
}

// TestRunBattle_AttackerWins 攻方先手击杀守方大营；守方战后伤兵+死亡
func TestRunBattle_AttackerWins(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{testSlot0(1001, 100, 50, 100, 80, 60, 5)}
	defender := []*pb_battle.TeamSlotInfo{testSlot0(2001, 100, 50, 100, 80, 60, 5)}

	r := runBattle(attacker, defender)

	// 攻方先手（同速攻方在前）：一击击穿守方大营（伤害 55×100 封顶 100）
	if len(r.GetRounds()) != 1 {
		t.Fatalf("期望 1 回合结束，实际 %d", len(r.GetRounds()))
	}
	act := r.GetRounds()[0].GetActions()
	if len(act) == 0 || act[0].GetActionType() != pb_battle.BattleActionType_BATTLE_ACTION_NORMAL_ATTACK {
		t.Fatalf("首行动期望普攻，实际 %+v", act)
	}
	// 第1回合受伤比例 85%：伤害100 → 死15 / 伤85
	if act[0].GetDamage() != 100 || act[0].GetKilled() != 15 || act[0].GetInjured() != 85 {
		t.Errorf("普攻 damage=%d killed=%d injured=%d, want 100/15/85", act[0].GetDamage(), act[0].GetKilled(), act[0].GetInjured())
	}
	// 结算阶段：85 伤兵 × 10% = 8 死亡
	if len(act) < 2 || act[len(act)-1].GetActionType() != pb_battle.BattleActionType_BATTLE_ACTION_SETTLEMENT ||
		act[len(act)-1].GetKilled() != 8 {
		t.Errorf("结算期望 killed=8，实际 %+v", act)
	}
	// 守方大营战后：有效0、伤兵 85-8=77
	if defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 0 {
		t.Errorf("守方大营有效兵力 = %d, want 0", defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum())
	}
	if defender[0].GetHeroInfo().GetSoldierInfo().GetCurInjuredNum() != 77 {
		t.Errorf("守方大营伤兵 = %d, want 77", defender[0].GetHeroInfo().GetSoldierInfo().GetCurInjuredNum())
	}
	// 攻方未承伤（守方大营已死无法行动）
	if attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 100 {
		t.Errorf("攻方大营有效兵力 = %d, want 100", attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum())
	}
	if r.GetAttacker().GetKilledSoldiers() != 23 { // 15死 + 8结算死
		t.Errorf("攻方击杀数 = %d, want 23", r.GetAttacker().GetKilledSoldiers())
	}
}

// TestRunBattle_Draw 双方仅大营且射程不足（<5）→ 互不可达，8 回合平局
func TestRunBattle_Draw(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{testSlot0(1001, 100, 50, 100, 80, 60, 3)}
	defender := []*pb_battle.TeamSlotInfo{testSlot0(2001, 100, 50, 100, 80, 60, 3)}

	r := runBattle(attacker, defender)

	if len(r.GetRounds()) != 8 {
		t.Fatalf("期望 8 回合平局，实际 %d", len(r.GetRounds()))
	}
	if attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 100 || defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 100 {
		t.Fatalf("平局双方大营均应有兵，实际 %d/%d", attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum(), defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum())
	}
}

// TestRunBattle_SameRoundMutual 同回合双方大营均归零 → 同归于尽
func TestRunBattle_SameRoundMutual(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{
		testSlot0(1001, 100, 50, 100, 80, 60, 5),          // 大营 先手
		testSlot(3,1002, 100, 30, 100, 80, 60, 5),        // 2号 后手
	}
	defender := []*pb_battle.TeamSlotInfo{
		testSlot0(2001, 100, 40, 100, 80, 60, 5),          // 大营
		testSlot(3,2002, 100, 35, 100, 80, 60, 5),        // 2号 先于攻方2号行动
	}

	r := runBattle(attacker, defender)

	// 行动序：攻大营(50) → 守大营(40) → 守2号(35) → 攻2号(30)
	// 攻大营击杀守2号；守大营反杀攻大营；攻2号补刀守大营 → 同归
	if attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 0 || defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum() != 0 {
		t.Fatalf("期望同归于尽（双方大营归零），实际 %d/%d", attacker[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum(), defender[0].GetHeroInfo().GetSoldierInfo().GetCurAliveNum())
	}
	if r.GetAttacker().GetKilledSoldiers() == 0 || r.GetDefender().GetKilledSoldiers() == 0 {
		t.Errorf("同归双方均应有击杀，实际 %d/%d", r.GetAttacker().GetKilledSoldiers(), r.GetDefender().GetKilledSoldiers())
	}
}

// TestRunBattle_PursuitSkill 追击技能按概率触发（重置计数器确保触发）
func TestRunBattle_PursuitSkill(t *testing.T) {
	_stepCounter = 0
	attacker := []*pb_battle.TeamSlotInfo{
		testSlot0(1001, 100, 50, 100, 80, 60, 5, &pb_skill.Skill{ConfigId: 102}), // 追击技能(触发30%)
	}
	defender := []*pb_battle.TeamSlotInfo{testSlot0(2001, 100000, 40, 100, 80, 60, 5)} // 兵多扛两下

	r := runBattle(attacker, defender)

	var hasPursuit bool
	for _, a := range r.GetRounds()[0].GetActions() {
		if a.GetActionType() == pb_battle.BattleActionType_BATTLE_ACTION_PURSUIT {
			hasPursuit = true
			if a.GetSkillConfId() != 102 {
				t.Errorf("追击技能配置 = %d, want 102", a.GetSkillConfId())
			}
		}
	}
	if !hasPursuit {
		t.Fatalf("期望追击技能触发（重置计数器后首掷必中），行动：%+v", r.GetRounds()[0].GetActions())
	}
}

// TestRunBattle_NormalAttackRange 射程按站位距离：攻大营(index0) rng4 够不到守大营(index5)；
// 攻2号位(index2) rng3 距离守大营(距离3) → 可攻击
func TestRunBattle_NormalAttackRange(t *testing.T) {
	defender := []*pb_battle.TeamSlotInfo{testSlot0(2001, 100, 40, 100, 80, 60, 5)}

	// 射程不足：双方大营 rng4 < 距离对方大营 5 → 全程无攻击行动
	attackerFar := []*pb_battle.TeamSlotInfo{testSlot0(1001, 100, 50, 100, 80, 60, 4)}
	defenderFar := []*pb_battle.TeamSlotInfo{testSlot0(2001, 100, 40, 100, 80, 60, 4)}
	r := runBattle(attackerFar, defenderFar)
	for _, br := range r.GetRounds() {
		for _, a := range br.GetActions() {
			if isAttackAction(a) {
				t.Fatalf("射程不足不应有攻击行动，实际 %+v", a)
			}
		}
	}

	// 射程足够：攻2号位 rng3（距离守大营 3）→ 能打到守大营
	attackerNear := []*pb_battle.TeamSlotInfo{testSlot(3, 1001, 100, 50, 100, 80, 60, 3)}
	r2 := runBattle(attackerNear, defender)
	hit := false
	for _, br := range r2.GetRounds() {
		for _, a := range br.GetActions() {
			if isAttackAction(a) && a.GetTargetSlot() == 1 { // 大营
				hit = true
			}
		}
	}
	if !hit {
		t.Fatalf("2号位 rng3 应能打到守方大营(距离3)，实际无攻击行动")
	}
}

// isAttackAction 是否攻击类行动（普攻/主动技能/追击）
func isAttackAction(a *pb_battle.BattleAction) bool {
	switch a.GetActionType() {
	case pb_battle.BattleActionType_BATTLE_ACTION_NORMAL_ATTACK,
		pb_battle.BattleActionType_BATTLE_ACTION_ACTIVE_SKILL,
		pb_battle.BattleActionType_BATTLE_ACTION_PURSUIT:
		return true
	}
	return false
}
