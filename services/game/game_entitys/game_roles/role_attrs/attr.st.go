package role_attrs

import (
	"server.slg.com/services/game/game_models"
)

// RoleAttrs 角色属性子模块（RoleID 1:1 单行，无 List/Mem 索引）
type RoleAttrs struct {
	RoleID uint64                `json:"role_id"`
	Attr   *game_models.RoleAttr `json:"attr,omitempty"`
}
