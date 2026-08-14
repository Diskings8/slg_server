package game_models

import (
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/models"
)

const hero_skill_collection = "hero_skill_collection"

// HeroSkillCollection 英雄技能收藏
type HeroSkillCollection struct {
	models.ModelBase
	RoleID          uint64               `gorm:"column:role_id;type:bigint(20);not null;comment:角色id;uniqueIndex:idx_hero_skill_collection,priority:1"`
	SkillConfID     int32                `gorm:"column:skill_conf_id;type:int(11);not null;comment:配置id;uniqueIndex:idx_hero_skill_collection,priority:2"`
	IsUnlocked      bool                 `gorm:"column:is_unlocked;type:tinyint(1);not null;comment:是否解锁;default:0"`
	CollectionLevel []*pb_common.Int32KV `gorm:"column:collection_level;serializer:jsonslice;type:json;not null;comment:收集进度"`
}

func (HeroSkillCollection) TableName() string {
	return hero_skill_collection
}
