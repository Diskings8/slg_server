package role_unions

import (
	"server.slg.com/services/game/game_models"
)

// RoleUnion 角色侧联盟快照实体（1:1，无 Mem 索引），包装 game_models.RoleUnion
type RoleUnion struct {
	*game_models.RoleUnion
}

// RoleUnions 角色下的联盟快照子模块（单行）
type RoleUnions struct {
	Union  *game_models.RoleUnion `json:"union,omitempty"`
	RoleID uint64                 `json:"role_id"`
}
