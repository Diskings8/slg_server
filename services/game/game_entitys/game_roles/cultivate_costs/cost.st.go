package cultivate_costs

import (
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/game/game_models"
)

// CultivateCost 养成消耗实体，包装 game_models.CultivateCost
type CultivateCost struct {
	*game_models.CultivateCost
}

// CultivateCosts 角色下的养成消耗集合
type CultivateCosts struct {
	List   []*game_models.CultivateCost                  `json:"list"`
	Mem    hashmaps.Map[pb_confs.ItemID, *CultivateCost] `json:"-"`
	RoleID uint64                                        `json:"role_id"`
}
