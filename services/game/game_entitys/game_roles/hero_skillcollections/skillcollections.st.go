package hero_skillcollections

import (
	"server.slg.com/services/game/game_models"
)

// HeroSkillCollection 英雄技能收藏条目
type HeroSkillCollection struct {
	*game_models.HeroSkillCollection
}

// HeroSkillCollections 玩家的所有英雄技能收藏
type HeroSkillCollections struct {
	List   []*game_models.HeroSkillCollection `json:"list"`
	RoleID uint64                             `json:"role_id"`
}

func NewHeroSkillCollections(roleID uint64) *HeroSkillCollections {
	return &HeroSkillCollections{
		RoleID: roleID,
		List:   make([]*game_models.HeroSkillCollection, 0),
	}
}

func (hsc *HeroSkillCollections) TableName() string {
	return "hero_skill_collections"
}
