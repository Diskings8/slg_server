package troop

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// NewFromPB 从 tabtoy 单一配置构建兵种配置（troop 表单行 + transform_cost）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Troop) == 0 {
		return nil, fmt.Errorf("troop table empty")
	}
	pb := t.Troop[0]
	c := &Conf{
		TransformLevel: pb.TransformLevel,
		DefaultTroopID: pb.DefaultTroopId,
		UnlockItemConf: pb.UnlockItemConf,
		TransformCost:  pbCosts(pb.TransformCost),
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("troop validate: %w", err)
	}
	return c, nil
}

// New 构造兵种配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}

// pbCosts 将 pb cost 列表转换为领域 ItemUse（与 JSON 路径 toCosts 同构）。
func pbCosts(costs []*pb_gameconfig.Cost) []common_declarations.ItemUse {
	if len(costs) == 0 {
		return nil
	}
	out := make([]common_declarations.ItemUse, 0, len(costs))
	for _, c := range costs {
		out = append(out, common_declarations.ItemUse{
			ItemID:   pb_confs.ItemID(c.ItemId),
			ItemType: pb_confs.ItemType(c.ItemType),
			Count:    c.Count,
		})
	}
	return out
}
