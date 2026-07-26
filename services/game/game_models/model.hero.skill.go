package game_models

import "server.slg.com/common/models"

const hero_skill = "hero_skill"

// HeroSkill 英雄技能
type HeroSkill struct {
	models.ModelBase
	RoleID        uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_hero_skill"`
	SkillConfID   int32  `gorm:"column:skill_conf_id;type:int(11);not null;uniqueIndex:idx_hero_skill"`
	Level         int32  `gorm:"column:level;type:int(11);not null;default:1"`
	IsAwakened    bool   `gorm:"column:is_awakened;type:tinyint(1);not null;default:0"`
	IsUnlocked    bool   `gorm:"column:is_unlocked;type:tinyint(1);not null;default:0"`
	ResearchLevel int32  `gorm:"column:research_level;type:int(11);not null;default:1"`
}

func (HeroSkill) TableName() string {
	return hero_skill
}
