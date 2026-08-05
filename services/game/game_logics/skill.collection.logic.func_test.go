package game_logics

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// collectionRole 构造测试角色并造英雄卡（confID → 张数）
func collectionRole(t *testing.T, heroes map[int32]int64) *game_roles.Role {
	t.Helper()
	role := game_roles.NewTest(50001)
	for confID, n := range heroes {
		for i := int64(0); i < n; i++ {
			role.GetHeroes().AddHero(confID)
		}
	}
	return role
}

// TestSkillCollectionActivate_ProgressAndUnlock 分次消耗英雄卡，进度累积，全部达标解锁并发放技能
func TestSkillCollectionActivate_ProgressAndUnlock(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 5, 2: 3}) // 101 需 英雄1×5 + 英雄2×3
	hero1 := role.GetHeroes().GetHeroesByConf(1)
	hero2 := role.GetHeroes().GetHeroesByConf(2)

	// 消耗 2 张英雄1，进度未达标
	if err := SkillCollectionActivate(role, 101, hero1[0].GetID()); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if err := SkillCollectionActivate(role, 101, hero1[1].GetID()); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	c := role.GetSkillCollections().GetBySkillConfID(101)
	if c == nil {
		t.Fatal("collection not created")
	}
	if c.IsUnlocked {
		t.Fatal("should not unlock yet (partial progress)")
	}
	if got := collectionCollected(c.CollectionLevel, 1); got != 2 {
		t.Errorf("collected hero1 = %d, want 2", got)
	}

	// 补足英雄1 至 5/5（仍缺英雄2，不解锁）
	for _, h := range hero1[2:] {
		if err := SkillCollectionActivate(role, 101, h.GetID()); err != nil {
			t.Fatalf("activate failed: %v", err)
		}
	}
	if role.GetSkillCollections().GetBySkillConfID(101).IsUnlocked {
		t.Fatal("should not unlock yet (hero2 not full)")
	}

	// 收集 3 张英雄2 → 解锁 + 发放技能
	for _, h := range hero2 {
		if err := SkillCollectionActivate(role, 101, h.GetID()); err != nil {
			t.Fatalf("activate failed: %v", err)
		}
	}
	c = role.GetSkillCollections().GetBySkillConfID(101)
	if !c.IsUnlocked {
		t.Fatal("should unlock after all heroes collected")
	}

	// 英雄卡已被消耗（内存中查不到）
	if role.GetHeroes().GetHero(hero1[0].GetID()) != nil {
		t.Error("consumed hero1[0] should be removed")
	}
	if role.GetHeroes().GetHero(hero2[0].GetID()) != nil {
		t.Error("consumed hero2[0] should be removed")
	}

	// 技能发放闭环：技能库存在该技能且已解锁（可装配）
	skill := role.GetSkills().GetSkillByConfID(101)
	if skill == nil {
		t.Fatal("skill 101 not granted to skill library")
	}
	if !skill.GetIsUnlocked() {
		t.Error("skill 101 should be unlocked")
	}

	// 养成消耗记录：8 次激活各记一条（5×英雄1 / 3×英雄2）
	costs := role.GetCultivateCosts().List
	if len(costs) != 8 {
		t.Fatalf("cultivate cost count = %d, want 8", len(costs))
	}
	if costs[0].CultivateType != pb_cultivate.CultivateType_CultivateSkill {
		t.Errorf("cultivate type = %v, want CultivateSkill", costs[0].CultivateType)
	}
	if len(costs[0].Cost) != 1 || costs[0].Cost[0].GetKey() != 1 || costs[0].Cost[0].GetVal() != 1 {
		t.Errorf("first cultivate cost = %+v, want [{key:1 val:1}]", costs[0].Cost)
	}
}

// TestSkillCollectionActivate_Unlocked 已解锁不可再激活
func TestSkillCollectionActivate_Unlocked(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 5, 2: 3})
	for _, h := range role.GetHeroes().GetHeroesByConf(1) {
		SkillCollectionActivate(role, 101, h.GetID())
	}
	for _, h := range role.GetHeroes().GetHeroesByConf(2) {
		SkillCollectionActivate(role, 101, h.GetID())
	}

	role.GetHeroes().AddHero(1) // 解锁后再造一张
	extra := role.GetHeroes().GetHeroesByConf(1)[0]
	if err := SkillCollectionActivate(role, 101, extra.GetID()); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionUnlocked) {
		t.Fatalf("err = %v, want ErrSkillCollectionUnlocked", err)
	}
}

// TestSkillCollectionActivate_HeroInvalid 消耗非收藏所需英雄
func TestSkillCollectionActivate_HeroInvalid(t *testing.T) {
	role := collectionRole(t, map[int32]int64{3: 1}) // 3 不在 101 所需
	hero := role.GetHeroes().GetHeroesByConf(3)[0]
	if err := SkillCollectionActivate(role, 101, hero.GetID()); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionHeroInvalid) {
		t.Fatalf("err = %v, want ErrSkillCollectionHeroInvalid", err)
	}
}

// TestSkillCollectionActivate_HeroFull 该英雄已收集满，不可再消耗
func TestSkillCollectionActivate_HeroFull(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 6}) // 5 用于收满 + 1 额外
	heroes1 := role.GetHeroes().GetHeroesByConf(1)
	for _, h := range heroes1[:5] {
		SkillCollectionActivate(role, 101, h.GetID()) // 英雄1 需求 5，已满
	}
	if err := SkillCollectionActivate(role, 101, heroes1[5].GetID()); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionHeroFull) {
		t.Fatalf("err = %v, want ErrSkillCollectionHeroFull", err)
	}
}

// TestSkillCollectionActivate_ConfNotFound 配置不存在
func TestSkillCollectionActivate_ConfNotFound(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 1})
	hero := role.GetHeroes().GetHeroesByConf(1)[0]
	if err := SkillCollectionActivate(role, 9999, hero.GetID()); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionConfNotFound) {
		t.Fatalf("err = %v, want ErrSkillCollectionConfNotFound", err)
	}
}

// TestSkillCollectionActivate_NotConsumable 已投入养成（升级）的英雄卡不可消耗
func TestSkillCollectionActivate_NotConsumable(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 1})
	hero := role.GetHeroes().GetHeroesByConf(1)[0]
	hero.SetLevel(10) // 已升级 → 不可消耗
	if err := SkillCollectionActivate(role, 101, hero.GetID()); !assertErrorCode(t, err, pb_error_code.ErrorCode_HeroNoConsumeCard) {
		t.Fatalf("err = %v, want ErrHeroNoConsumeCard", err)
	}
}

// TestSkillCollectionActivate_HeroNotFound 英雄卡不存在
func TestSkillCollectionActivate_HeroNotFound(t *testing.T) {
	role := collectionRole(t, map[int32]int64{1: 1})
	if err := SkillCollectionActivate(role, 101, 999999); !assertErrorCode(t, err, pb_error_code.ErrorCode_ParamError) {
		t.Fatalf("err = %v, want ErrParamError", err)
	}
}
