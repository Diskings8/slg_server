package battle_logics

// 战斗经验结算单测 — 公式 / 平分 / 无参战 / 无击杀 / 敌方平均等级。

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"
)

func expHero(slotID int32, heroID uint64, alive uint32, level uint32) *pb_battle.TeamSlotInfo {
	return &pb_battle.TeamSlotInfo{
		SlotId: slotID,
		HeroInfo: &pb_hero.HeroInfo{
			ConfigId:  1,
			CurLevel:  level,
			HeroId:    heroID,
			SoldierInfo: &pb_hero.SoldierInfo{
				CurAliveNum: alive,
			},
		},
	}
}

// TestGrantBattleExp_Formula 公式：敌方平均等级 × 击杀敌兵 × 系数 ÷ 参战英雄
func TestGrantBattleExp_Formula(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{
		expHero(0, 1001, 100, 1),
		expHero(1, 1002, 100, 1),
	}
	defender := []*pb_battle.TeamSlotInfo{expHero(0, 2001, 50, 5)}
	result := &pb_battle.OneBattleResult{}

	// total = 5 × 40 × 5(coeff) = 1000；平分 2 人 → 500
	grantBattleExp(attacker, defender, 40, result)

	if len(result.GetHeroExp()) != 2 {
		t.Fatalf("期望 2 个参战英雄经验，实际 %d", len(result.GetHeroExp()))
	}
	for _, he := range result.GetHeroExp() {
		if he.GetExp() != 500 {
			t.Errorf("hero %d exp = %d, want 500", he.GetHeroId(), he.GetExp())
		}
	}
}

// TestGrantBattleExp_Split 经验平分给存活参战英雄
func TestGrantBattleExp_Split(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{
		expHero(0, 1001, 100, 1),
		expHero(1, 1002, 100, 1),
		expHero(2, 1003, 100, 1),
		expHero(3, 1004, 0, 1), // 无兵 → 不参战
	}
	defender := []*pb_battle.TeamSlotInfo{expHero(0, 2001, 50, 10)}
	result := &pb_battle.OneBattleResult{}

	// total = 10 × 40 × 5 = 2000；÷3 = 666
	grantBattleExp(attacker, defender, 40, result)

	if len(result.GetHeroExp()) != 3 {
		t.Fatalf("期望 3 个参战英雄经验（死兵不参战），实际 %d", len(result.GetHeroExp()))
	}
	for _, he := range result.GetHeroExp() {
		if he.GetExp() != 666 {
			t.Errorf("hero %d exp = %d, want 666", he.GetHeroId(), he.GetExp())
		}
	}
}

// TestGrantBattleExp_NoParticipant 无存活参战英雄 → 无经验
func TestGrantBattleExp_NoParticipant(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{expHero(0, 1001, 0, 1)}
	defender := []*pb_battle.TeamSlotInfo{expHero(0, 2001, 50, 5)}
	result := &pb_battle.OneBattleResult{}

	grantBattleExp(attacker, defender, 40, result)

	if len(result.GetHeroExp()) != 0 {
		t.Fatalf("无存活参战英雄期望无经验，实际 %d", len(result.GetHeroExp()))
	}
}

// TestGrantBattleExp_NoKill 无击杀敌兵 → 无经验
func TestGrantBattleExp_NoKill(t *testing.T) {
	attacker := []*pb_battle.TeamSlotInfo{expHero(0, 1001, 100, 1)}
	defender := []*pb_battle.TeamSlotInfo{expHero(0, 2001, 50, 5)}
	result := &pb_battle.OneBattleResult{}

	grantBattleExp(attacker, defender, 0, result) // killedEnemy=0

	if len(result.GetHeroExp()) != 0 {
		t.Fatalf("无击杀期望无经验，实际 %d", len(result.GetHeroExp()))
	}
}

// TestTeamAttack 属性加权攻击力 = 攻击 × 存活士兵（快照组件求和）
func TestTeamAttack(t *testing.T) {
	slots := []*pb_battle.TeamSlotInfo{testSlot(0, 1001, 100, 50, 100, 80, 60, 5)}
	// 攻击 100 × 100 = 10000
	if got := teamAttack(slots); got != 10000 {
		t.Errorf("teamAttack = %d, want 10000", got)
	}
	// 防御 80 × 100 = 8000
	if got := teamDefense(slots); got != 8000 {
		t.Errorf("teamDefense = %d, want 8000", got)
	}
}
