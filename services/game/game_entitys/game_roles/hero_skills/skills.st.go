package hero_skills

import (
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// HeroSkill 玩家技能库中的一条技能记录
type HeroSkill struct {
	*game_models.HeroSkill
}

// HeroSkills 玩家英雄技能库，按 SkillConfID 索引
type HeroSkills struct {
	List   []*game_models.HeroSkill                  `json:"list"`
	Mem    hashmaps.Map[pb_confs.ItemID, *HeroSkill] `json:"-"`
	RoleID uint64                                    `json:"role_id"`
}
