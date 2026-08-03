package role_formations

import (
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// RoleFormation 已上阵队伍（队列）实体，包装 game_models.RoleFormation
type RoleFormation struct {
	*game_models.RoleFormation
}

// RoleFormations 角色下所有城市的队列
// Mem key = 唯一队列ID
type RoleFormations struct {
	List   []*game_models.RoleFormation         `json:"list"`
	Mem    hashmaps.Map[uint64, *RoleFormation] `json:"-"`
	RoleID uint64                               `json:"role_id"`
}
