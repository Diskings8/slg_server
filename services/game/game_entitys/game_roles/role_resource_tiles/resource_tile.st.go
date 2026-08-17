package role_resource_tiles

import (
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// RoleResourceTile 资源地产出快照实体，包装 game_models.RoleResourceTile
type RoleResourceTile struct {
	*game_models.RoleResourceTile
}

// RoleResourceTiles 角色下所有资源地产出快照
// Mem key = 地块 MapID（同一地块同一角色唯一）
type RoleResourceTiles struct {
	List   []*game_models.RoleResourceTile         `json:"list"`
	Mem    hashmaps.Map[int32, *RoleResourceTile]  `json:"-"`
	RoleID uint64                                  `json:"role_id"`
}
