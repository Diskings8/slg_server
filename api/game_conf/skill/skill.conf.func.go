package skill

import (
	"fmt"
)

// AllSkills 全部技能配置（供跨表校验遍历）。
func (c *Conf) AllSkills() []SkillConf { return c.skillList }

// AllCollections 全部技能收藏配置（供跨表校验遍历）。
func (c *Conf) AllCollections() []SkillCollectionConf {
	list := make([]SkillCollectionConf, 0, len(c.collectionConfs))
	for _, cc := range c.collectionConfs {
		list = append(list, cc)
	}
	return list
}

// Validate 校验技能配置完整性（主键唯一/枚举白名单/表内引用/槽位约束）
func (c *Conf) Validate() error {
	if len(c.skillByID) == 0 {
		return fmt.Errorf("skills must not be empty")
	}
	for id, s := range c.skillByID {
		if id <= 0 {
			return fmt.Errorf("skill conf_id must be > 0, got %d", id)
		}
		if s.MaxLevel <= 0 {
			return fmt.Errorf("skill %d max_level must be > 0", id)
		}
		if s.UseLimit < 0 {
			return fmt.Errorf("skill %d use_limit must be >= 0", id)
		}
		if s.TriggerRate > 100 {
			return fmt.Errorf("skill %d trigger_rate must be <= 100, got %d", id, s.TriggerRate)
		}
		switch s.SkillType {
		case SkillTypeActive, SkillTypePursuit, SkillTypePassive:
		default:
			return fmt.Errorf("skill %d invalid skill_type %d", id, s.SkillType)
		}
		switch s.TargetType {
		case TargetRandom, TargetFront, TargetBase:
		default:
			return fmt.Errorf("skill %d invalid target_type %d", id, s.TargetType)
		}
		switch s.EffectType {
		case EffectPhysDamage, EffectMagicDamage, EffectRecover:
		default:
			return fmt.Errorf("skill %d invalid effect_type %d", id, s.EffectType)
		}
		if s.UpgradeCost.Count <= 0 {
			return fmt.Errorf("skill %d upgrade_cost count must be > 0", id)
		}
	}
	// 收藏：skill_conf_id 须存在于技能表
	for skillID, cc := range c.collectionConfs {
		if _, ok := c.skillByID[skillID]; !ok {
			return fmt.Errorf("collection skill_conf_id %d not found in skills", skillID)
		}
		if len(cc.NeedHeroes) == 0 {
			return fmt.Errorf("collection %d need_heroes must not be empty", skillID)
		}
		for _, h := range cc.NeedHeroes {
			if h.Count <= 0 {
				return fmt.Errorf("collection %d need_hero count must be > 0", skillID)
			}
		}
	}
	// 槽位约束
	if c.SlotEquipMin > c.SlotEquipMax {
		return fmt.Errorf("slot_equip_min %d > slot_equip_max %d", c.SlotEquipMin, c.SlotEquipMax)
	}
	if c.Slot1UnlockLv > c.Slot2UnlockLv {
		return fmt.Errorf("slot1_unlock_lv %d > slot2_unlock_lv %d", c.Slot1UnlockLv, c.Slot2UnlockLv)
	}
	if c.UnequipRefund <= 0 {
		return fmt.Errorf("unequip_refund must be > 0")
	}
	return nil
}
