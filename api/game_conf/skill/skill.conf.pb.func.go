package skill

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// NewFromPB 从 tabtoy 单一配置构建技能配置（skill + skill_collection + skill_setting 三表拼装）。
//
// 迁移原 Load 的「局部构建 + 末尾提交 + 校验」：构造失败返回 err，不产生半更新。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	skillList := make([]SkillConf, 0, len(t.Skill))
	skillByID := make(map[int32]SkillConf, len(t.Skill))
	for _, row := range t.Skill {
		if _, dup := skillByID[row.ConfId]; dup {
			return nil, fmt.Errorf("duplicate skill conf_id %d", row.ConfId)
		}
		upgrade := common_declarations.ItemUse{}
		if row.UpgradeCost != nil {
			upgrade = common_declarations.ItemUse{
				ItemID:   pb_confs.ItemID(row.UpgradeCost.ItemId),
				ItemType: pb_confs.ItemType(row.UpgradeCost.ItemType),
				Count:    row.UpgradeCost.Count,
			}
		}
		s := SkillConf{
			ConfID:        row.ConfId,
			MaxLevel:      row.MaxLevel,
			UseLimit:      row.UseLimit,
			UpgradeCost:   upgrade,
			SkillType:     SkillType(row.SkillType),
			TargetType:    TargetType(row.TargetType),
			EffectType:    EffectType(row.EffectType),
			DamageCoeff:   row.DamageCoeff,
			ConvergeCoeff: row.ConvergeCoeff,
			TriggerRate:   row.TriggerRate,
		}
		skillList = append(skillList, s)
		skillByID[row.ConfId] = s
	}

	// 收藏：所需英雄卡（pb reward → ItemUse，ItemType 恒 0）
	collectionConfs := make(map[int32]SkillCollectionConf, len(t.SkillCollection))
	for _, row := range t.SkillCollection {
		if _, dup := collectionConfs[row.SkillConfId]; dup {
			return nil, fmt.Errorf("duplicate collection skill_conf_id %d", row.SkillConfId)
		}
		need := make([]common_declarations.ItemUse, 0, len(row.NeedHeroes))
		for _, h := range row.NeedHeroes {
			need = append(need, common_declarations.ItemUse{ItemID: pb_confs.ItemID(h.ItemId), Count: h.Count})
		}
		collectionConfs[row.SkillConfId] = SkillCollectionConf{SkillConfID: row.SkillConfId, NeedHeroes: need}
	}

	c := &Conf{
		skillList:       skillList,
		skillByID:       skillByID,
		collectionConfs: collectionConfs,
	}
	if len(t.SkillSetting) == 0 {
		return nil, fmt.Errorf("skill_setting table empty")
	}
	ss := t.SkillSetting[0]
	c.SlotDefault = ss.SlotDefault
	c.SlotEquipMin = ss.SlotEquipMin
	c.SlotEquipMax = ss.SlotEquipMax
	c.Slot1UnlockLv = ss.Slot1UnlockLv
	c.Slot2UnlockLv = ss.Slot2UnlockLv
	c.UnequipRefund = ss.UnequipRefund

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("skill validate: %w", err)
	}
	return c, nil
}

// New 构造技能配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
