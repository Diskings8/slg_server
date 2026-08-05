package skill

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
)

// itemUseJSON 域内 ItemUse 的 JSON 镜像（common_declarations.ItemUse 无 json tag，需中间结构转换）。
type itemUseJSON struct {
	ItemID   int32 `json:"item_id"`
	ItemType int32 `json:"item_type,omitempty"`
	Count    int64 `json:"count"`
}

func (j itemUseJSON) toItemUse() common_declarations.ItemUse {
	return common_declarations.ItemUse{
		ItemID:   pb_confs.ItemID(j.ItemID),
		ItemType: pb_confs.ItemType(j.ItemType),
		Count:    j.Count,
	}
}

// skillJSON 技能配置表 JSON 结构（磁盘格式，snake_case）
type skillJSON struct {
	Skills        []skillRowJSON      `json:"skills"`
	Collections   []collectionRowJSON `json:"collections"`
	SlotDefault   int32               `json:"slot_default"`
	SlotEquipMin  int32               `json:"slot_equip_min"`
	SlotEquipMax  int32               `json:"slot_equip_max"`
	Slot1UnlockLv uint32              `json:"slot1_unlock_lv"`
	Slot2UnlockLv uint32              `json:"slot2_unlock_lv"`
	UnequipRefund int32               `json:"unequip_refund"`
}

// skillRowJSON 单技能配置行
type skillRowJSON struct {
	ConfID        int32       `json:"conf_id"`
	MaxLevel      int32       `json:"max_level"`
	UseLimit      int32       `json:"use_limit"`
	UpgradeCost   itemUseJSON `json:"upgrade_cost"`
	SkillType     SkillType   `json:"skill_type"`
	TargetType    TargetType  `json:"target_type"`
	EffectType    EffectType  `json:"effect_type"`
	DamageCoeff   uint32      `json:"damage_coeff"`
	ConvergeCoeff uint32      `json:"converge_coeff"`
	TriggerRate   uint32      `json:"trigger_rate"`
}

// collectionRowJSON 单技能收藏配置行
type collectionRowJSON struct {
	SkillConfID int32         `json:"skill_conf_id"`
	NeedHeroes  []itemUseJSON `json:"need_heroes"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "skill" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

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

// Load 从 JSON 字节构建技能配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j skillJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	skillList := make([]SkillConf, 0, len(j.Skills))
	skillByID := make(map[int32]SkillConf, len(j.Skills))
	for _, row := range j.Skills {
		if _, dup := skillByID[row.ConfID]; dup {
			return fmt.Errorf("duplicate skill conf_id %d", row.ConfID)
		}
		s := SkillConf{
			ConfID:        row.ConfID,
			MaxLevel:      row.MaxLevel,
			UseLimit:      row.UseLimit,
			UpgradeCost:   row.UpgradeCost.toItemUse(),
			SkillType:     row.SkillType,
			TargetType:    row.TargetType,
			EffectType:    row.EffectType,
			DamageCoeff:   row.DamageCoeff,
			ConvergeCoeff: row.ConvergeCoeff,
			TriggerRate:   row.TriggerRate,
		}
		skillList = append(skillList, s)
		skillByID[row.ConfID] = s
	}

	collectionConfs := make(map[int32]SkillCollectionConf, len(j.Collections))
	for _, row := range j.Collections {
		if _, dup := collectionConfs[row.SkillConfID]; dup {
			return fmt.Errorf("duplicate collection skill_conf_id %d", row.SkillConfID)
		}
		need := make([]common_declarations.ItemUse, 0, len(row.NeedHeroes))
		for _, h := range row.NeedHeroes {
			need = append(need, h.toItemUse())
		}
		collectionConfs[row.SkillConfID] = SkillCollectionConf{
			SkillConfID: row.SkillConfID,
			NeedHeroes:  need,
		}
	}

	// 末尾一次性提交，保证失败不产生半更新
	c.skillList = skillList
	c.skillByID = skillByID
	c.collectionConfs = collectionConfs
	c.SlotDefault = j.SlotDefault
	c.SlotEquipMin = j.SlotEquipMin
	c.SlotEquipMax = j.SlotEquipMax
	c.Slot1UnlockLv = j.Slot1UnlockLv
	c.Slot2UnlockLv = j.Slot2UnlockLv
	c.UnequipRefund = j.UnequipRefund
	c.version = table.ContentHash(data)
	return nil
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
