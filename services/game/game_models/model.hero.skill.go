package game_models

import "server.slg.com/common/models"

const hero_skill = "hero_skill"

// HeroSkill 英雄技能
type HeroSkill struct {
	models.ModelBase
	RoleID        uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_hero_skill,priority:1;comment:角色ID"`
	SkillConfID   int32  `gorm:"column:skill_conf_id;type:int(11);not null;uniqueIndex:idx_hero_skill,priority:2;comment:技能配置ID"`
	Level         int32  `gorm:"column:level;type:int(11);not null;default:1;comment:技能等级"`
	IsAwakened    bool   `gorm:"column:is_awakened;type:tinyint(1);not null;default:0;comment:是否觉醒"`
	IsUnlocked    bool   `gorm:"column:is_unlocked;type:tinyint(1);not null;default:0;comment:是否解锁"`
	ResearchLevel int32  `gorm:"column:research_level;type:int(11);not null;default:1;comment:研究等级"`
	// EquipHeroID 当前被装配的英雄实例ID（0=未装配；同一技能同时只能被一个英雄装配）
	EquipHeroID uint64 `gorm:"column:equip_hero_id;type:bigint(20);not null;default:0;comment:当前被装配的英雄实例ID(0=未装配)"`
	// UseCountLimit 可装配次数上限（技能可被英雄装配/使用的总次数）
	UseCountLimit int32 `gorm:"column:use_count_limit;type:int(11);not null;default:0;comment:可装配次数上限"`
	// UsedCount 已装配次数（每次装配 +1，满 UseCountLimit 不可再装配）
	UsedCount int32 `gorm:"column:used_count;type:int(11);not null;default:0;comment:已装配次数"`
}

func (HeroSkill) TableName() string {
	return hero_skill
}
