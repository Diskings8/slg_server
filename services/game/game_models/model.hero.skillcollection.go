package game_models

import "server.slg.com/common/models"

const hero_skill_collection = "hero_skill_collection"

// HeroSkillCollection 英雄技能收藏
type HeroSkillCollection struct {
	models.ModelBase
	RoleID          uint64  `gorm:"column:role_id;type:bigint(20);not null;comment:角色id;uniqueIndex:idx_hero_skill"`
	SkillConfID     int32   `gorm:"column:skill_conf_id;type:int(11);not null;comment:配置id;uniqueIndex:idx_hero_skill"`
	IsUnlocked      bool    `gorm:"column:is_unlocked;type:tinyint(1);not null;comment:是否解锁;default:0"`
	CollectionLevel []int32 `gorm:"column:collection_level;type:json;not null;comment:收集进度"`
}

func (HeroSkillCollection) TableName() string {
	return hero_skill_collection
}
