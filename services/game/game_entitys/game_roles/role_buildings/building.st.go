package role_buildings

import (
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// RoleBuilding 角色建筑实体，包装 game_models.RoleBuilding
// 主城/分城/军事建筑统一，type 字段区分
type RoleBuilding struct {
	*game_models.RoleBuilding
}

// RoleBuildings 角色下的所有建筑集合
type RoleBuildings struct {
	List   []*game_models.RoleBuilding          `json:"list"`
	Mem    hashmaps.Map[uint64, *RoleBuilding]  `json:"-"`
	RoleID uint64                               `json:"role_id"`
}
