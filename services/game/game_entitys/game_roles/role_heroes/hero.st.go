package role_heroes

import (
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// RoleHero 英雄实体，包装 game_models.RoleHero
type RoleHero struct {
	*game_models.RoleHero
}

// RoleHeroes 角色下的所有英雄集合
type RoleHeroes struct {
	List   []*game_models.RoleHero       `json:"list"`
	Mem    hashmaps.Map[uint64, *RoleHero] `json:"-"`
	RoleID uint64                        `json:"role_id"`
}
