package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSkill_LoadAndQuery JSON 加载后技能/收藏/槽位查询正常。
func TestSkill_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "skills": [
    {"conf_id": 101, "max_level": 10, "use_limit": 3, "upgrade_cost": {"item_id": 2001, "count": 1}, "skill_type": 1, "target_type": 1, "effect_type": 1, "damage_coeff": 100, "converge_coeff": 0, "trigger_rate": 0}
  ],
  "collections": [
    {"skill_conf_id": 101, "need_heroes": [{"item_id": 1, "count": 5}, {"item_id": 2, "count": 3}]}
  ],
  "slot_default": 0, "slot_equip_min": 1, "slot_equip_max": 2,
  "slot1_unlock_lv": 10, "slot2_unlock_lv": 20, "unequip_refund": 2
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if c.FileName() != "skill" {
		t.Errorf("FileName = %q, want skill", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
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

// TestSkill_LoadDuplicateKey 技能主键重复 → Load 报错。
func TestSkill_LoadDuplicateKey(t *testing.T) {
	c := New()
	data := []byte(`{
  "skills": [
    {"conf_id": 101, "max_level": 10, "use_limit": 3, "upgrade_cost": {"item_id": 2001, "count": 1}, "skill_type": 1, "target_type": 1, "effect_type": 1, "damage_coeff": 100},
    {"conf_id": 101, "max_level": 5, "use_limit": 1, "upgrade_cost": {"item_id": 2002, "count": 1}, "skill_type": 1, "target_type": 3, "effect_type": 2, "damage_coeff": 120}
  ],
  "collections": [], "slot_default": 0, "slot_equip_min": 1, "slot_equip_max": 2,
  "slot1_unlock_lv": 10, "slot2_unlock_lv": 20, "unequip_refund": 2
}`)
	if err := c.Load(data); err == nil {
		t.Fatal("Load should fail on duplicate conf_id")
	}
}

// TestSkill_ValidateInvalidEnum 非法枚举 → Validate 报错。
func TestSkill_ValidateInvalidEnum(t *testing.T) {
	c := New()
	data := []byte(`{
  "skills": [
    {"conf_id": 101, "max_level": 10, "use_limit": 3, "upgrade_cost": {"item_id": 2001, "count": 1}, "skill_type": 99, "target_type": 1, "effect_type": 1, "damage_coeff": 100}
  ],
  "collections": [], "slot_default": 0, "slot_equip_min": 1, "slot_equip_max": 2,
  "slot1_unlock_lv": 10, "slot2_unlock_lv": 20, "unequip_refund": 2
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail on invalid skill_type")
	}
}

// TestSkill_ValidateCollectionRefMissing 收藏引用了不存在的技能 → Validate 报错。
func TestSkill_ValidateCollectionRefMissing(t *testing.T) {
	c := New()
	data := []byte(`{
  "skills": [
    {"conf_id": 101, "max_level": 10, "use_limit": 3, "upgrade_cost": {"item_id": 2001, "count": 1}, "skill_type": 1, "target_type": 1, "effect_type": 1, "damage_coeff": 100}
  ],
  "collections": [{"skill_conf_id": 999, "need_heroes": [{"item_id": 1, "count": 5}]}],
  "slot_default": 0, "slot_equip_min": 1, "slot_equip_max": 2,
  "slot1_unlock_lv": 10, "slot2_unlock_lv": 20, "unequip_refund": 2
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail on collection referencing missing skill")
	}
}

// TestSkill_RealJSONMatchesEmbedded 仓库 json/skill.json 与内嵌占位逐值一致。
func TestSkill_RealJSONMatchesEmbedded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "skill.json"))
	if err != nil {
		t.Skipf("skill.json not found, skip: %v", err)
	}
	embed, _ := New().GetSkillConf(101)
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real skill.json: %v", err)
	}
	got, _ := jc.GetSkillConf(101)
	if got != embed {
		t.Errorf("skill 101 json=%+v embedded=%+v", got, embed)
	}
}
