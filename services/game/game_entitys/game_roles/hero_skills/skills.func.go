package hero_skills

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
	"server.slg.com/common/models"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewHeroSkills(roleID uint64) *HeroSkills {
	return &HeroSkills{
		RoleID: roleID,
		List:   make([]*game_models.HeroSkill, 0),
	}
}

func (hss *HeroSkills) Init() {
	for _, modelOne := range hss.List {
		heroSkill := NewHeroSkill(modelOne)
		hss.Mem.Store(pb_confs.ItemID(heroSkill.ID), heroSkill)
	}
}

func (hss *HeroSkills) Copy(src *HeroSkills) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, hss)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	hss.Init()
}

func (hss *HeroSkills) Format2Pb() []*pb_skill.Skill {
	list := make([]*pb_skill.Skill, 0, len(hss.List))
	for _, v := range hss.List {
		item := NewHeroSkill(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

//-------------------------------

func NewHeroSkill(one *game_models.HeroSkill) *HeroSkill {
	h := &HeroSkill{
		HeroSkill: one,
	}
	// to add other attr
	return h
}

func (hs *HeroSkill) Format2Pb() *pb_skill.Skill {
	if hs.HeroSkill == nil {
		return nil
	}
	return &pb_skill.Skill{
		ConfigId:       hs.SkillConfID,
		Level:          hs.Level,
		Research_Level: hs.ResearchLevel,
		IsUnlock:       hs.IsUnlocked,
	}
}

//-------------------------------
// 技能库业务方法
//
// ⚠️ Mem 索引键是主键 ID（见 Init），技能库规模 ≤ 数十条，按 SkillConfID 查询直接遍历 List，
// 不新增第二索引，避免 Init/Copy/Add 双索引一致性问题。

// GetSkillByConfID 按配置ID查询技能（不存在返回 nil）
func (hss *HeroSkills) GetSkillByConfID(skillConfID int32) *HeroSkill {
	for _, modelOne := range hss.List {
		if modelOne.SkillConfID == skillConfID {
			return NewHeroSkill(modelOne)
		}
	}
	return nil
}

// AddSkill 新增技能到技能库（已存在返回 nil；新技能默认未解锁、等级1、可用装配次数 useCountLimit）
func (hss *HeroSkills) AddSkill(skillConfID, useCountLimit int32) *HeroSkill {
	if hss.GetSkillByConfID(skillConfID) != nil {
		return nil
	}
	now := time.Now().Unix()
	modelOne := &game_models.HeroSkill{
		ModelBase: models.ModelBase{
			ID:        snowflakes.GenUUID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		RoleID:        hss.RoleID,
		SkillConfID:   skillConfID,
		Level:         1,
		UseCountLimit: useCountLimit,
	}
	one := NewHeroSkill(modelOne)
	hss.List = append(hss.List, modelOne)
	hss.Mem.Store(pb_confs.ItemID(modelOne.ID), one) // 与 Init 的主键键一致
	return one
}

// UnlockSkill 幂等解锁：不存在则创建，已存在则置位。
//
// 返回技能 + 是否本次新解锁（用于差异化通知客户端）。
func (hss *HeroSkills) UnlockSkill(skillConfID, useCountLimit int32) (*HeroSkill, bool) {
	hs := hss.GetSkillByConfID(skillConfID)
	if hs == nil {
		hs = hss.AddSkill(skillConfID, useCountLimit)
	}
	if hs.GetIsUnlocked() {
		return hs, false
	}
	hs.SetIsUnlocked(true)
	return hs, true
}

//-------------------------------
// 装配状态

// EquipTo 装配到英雄：记录装配英雄 + 装配次数 +1
func (hs *HeroSkill) EquipTo(heroID uint64) {
	hs.SetEquipHeroID(heroID)
	hs.SetUsedCount(hs.GetUsedCount() + 1)
}

// Unequip 拆卸：清空装配英雄（不动次数，次数为累计消耗）
func (hs *HeroSkill) Unequip() {
	hs.SetEquipHeroID(0)
}
