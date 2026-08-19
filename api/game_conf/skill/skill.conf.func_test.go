package skill

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// TestSkill_LoadAndQuery 经 pb.Table → NewFromPB 构建后技能/收藏/槽位查询正常。
func TestSkill_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Skill: []*pb_gameconfig.Skill{
			{
				ConfId: 101, MaxLevel: 10, UseLimit: 3,
				UpgradeCost: &pb_gameconfig.Cost{ItemId: 2001, Count: 1},
				SkillType:   pb_gameconfig.Skilltype_skilltype_active,
				TargetType:  pb_gameconfig.Targettype_targettype_random,
				EffectType:  pb_gameconfig.Effecttype_effecttype_phys_damage,
				DamageCoeff: 100,
			},
		},
		SkillCollection: []*pb_gameconfig.SkillCollection{
			{
				SkillConfId: 101,
				NeedHeroes:  []*pb_gameconfig.Reward{{ItemId: 1, Count: 5}, {ItemId: 2, Count: 3}},
			},
		},
		SkillSetting: []*pb_gameconfig.SkillSetting{
			{SlotDefault: 0, SlotEquipMin: 1, SlotEquipMax: 2, Slot1UnlockLv: 10, Slot2UnlockLv: 20, UnequipRefund: 2},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	s, ok := c.GetSkillConf(101)
	if !ok || s.MaxLevel != 10 || s.UpgradeCost.ItemID != 2001 || s.UpgradeCost.Count != 1 {
		t.Errorf("GetSkillConf(101) = %+v, ok=%v", s, ok)
	}
	if s.SkillType != SkillTypeActive || s.TargetType != TargetRandom || s.EffectType != EffectPhysDamage {
		t.Errorf("skill 101 enums = %d/%d/%d", s.SkillType, s.TargetType, s.EffectType)
	}
	cc, ok := c.GetCollectionConf(101)
	if !ok || len(cc.NeedHeroes) != 2 || cc.NeedHeroes[0].ItemID != 1 {
		t.Errorf("GetCollectionConf(101) = %+v, ok=%v", cc, ok)
	}
	if c.SlotEquipMin != 1 || c.Slot1UnlockLv != 10 || c.UnequipRefund != 2 {
		t.Errorf("slots = %d/%d/%d", c.SlotEquipMin, c.Slot1UnlockLv, c.UnequipRefund)
	}
}

// TestSkill_LoadDuplicateKey 技能主键重复 → NewFromPB 报错。
func TestSkill_LoadDuplicateKey(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Skill: []*pb_gameconfig.Skill{
			{ConfId: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: &pb_gameconfig.Cost{ItemId: 2001, Count: 1},
				SkillType: pb_gameconfig.Skilltype_skilltype_active, TargetType: pb_gameconfig.Targettype_targettype_random,
				EffectType: pb_gameconfig.Effecttype_effecttype_phys_damage, DamageCoeff: 100},
			{ConfId: 101, MaxLevel: 5, UseLimit: 1, UpgradeCost: &pb_gameconfig.Cost{ItemId: 2002, Count: 1},
				SkillType: pb_gameconfig.Skilltype_skilltype_active, TargetType: pb_gameconfig.Targettype_targettype_base,
				EffectType: pb_gameconfig.Effecttype_effecttype_magic_damage, DamageCoeff: 120},
		},
		SkillSetting: []*pb_gameconfig.SkillSetting{
			{SlotDefault: 0, SlotEquipMin: 1, SlotEquipMax: 2, Slot1UnlockLv: 10, Slot2UnlockLv: 20, UnequipRefund: 2},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on duplicate conf_id")
	}
}

// TestSkill_ValidateInvalidEnum 非法枚举 → NewFromPB 校验报错。
func TestSkill_ValidateInvalidEnum(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Skill: []*pb_gameconfig.Skill{
			{ConfId: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: &pb_gameconfig.Cost{ItemId: 2001, Count: 1},
				SkillType: 99, TargetType: pb_gameconfig.Targettype_targettype_random,
				EffectType: pb_gameconfig.Effecttype_effecttype_phys_damage, DamageCoeff: 100},
		},
		SkillSetting: []*pb_gameconfig.SkillSetting{
			{SlotDefault: 0, SlotEquipMin: 1, SlotEquipMax: 2, Slot1UnlockLv: 10, Slot2UnlockLv: 20, UnequipRefund: 2},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on invalid skill_type")
	}
}

// TestSkill_ValidateCollectionRefMissing 收藏引用了不存在的技能 → NewFromPB 校验报错。
func TestSkill_ValidateCollectionRefMissing(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Skill: []*pb_gameconfig.Skill{
			{ConfId: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: &pb_gameconfig.Cost{ItemId: 2001, Count: 1},
				SkillType: pb_gameconfig.Skilltype_skilltype_active, TargetType: pb_gameconfig.Targettype_targettype_random,
				EffectType: pb_gameconfig.Effecttype_effecttype_phys_damage, DamageCoeff: 100},
		},
		SkillCollection: []*pb_gameconfig.SkillCollection{
			{SkillConfId: 999, NeedHeroes: []*pb_gameconfig.Reward{{ItemId: 1, Count: 5}}},
		},
		SkillSetting: []*pb_gameconfig.SkillSetting{
			{SlotDefault: 0, SlotEquipMin: 1, SlotEquipMax: 2, Slot1UnlockLv: 10, Slot2UnlockLv: 20, UnequipRefund: 2},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on collection referencing missing skill")
	}
}
