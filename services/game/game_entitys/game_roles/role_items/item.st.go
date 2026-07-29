package role_items

import (
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// RoleItem 道具实体，包装 game_models.RoleItem
type RoleItem struct {
	*game_models.RoleItem
}

// RoleItems 角色下的所有道具集合
type RoleItems struct {
	List   []*game_models.RoleItem                  `json:"list"`
	Mem    hashmaps.Map[pb_confs.ItemID, *RoleItem] `json:"-"`
	RoleID uint64                                   `json:"role_id"`
}
