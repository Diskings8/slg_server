package item

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
)

// NewFromPB 从 tabtoy 单一配置构建道具配置（item 表逐行 → configs 索引）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	configs := make(map[pb_confs.ItemID]ItemConfig, len(t.Item))
	for _, row := range t.Item {
		if row.ConfId <= 0 {
			return nil, fmt.Errorf("item conf_id must be > 0, got %d", row.ConfId)
		}
		if _, dup := configs[pb_confs.ItemID(row.ConfId)]; dup {
			return nil, fmt.Errorf("duplicate item conf_id %d", row.ConfId)
		}
		configs[pb_confs.ItemID(row.ConfId)] = ItemConfig{
			ConfID: pb_confs.ItemID(row.ConfId),
			Effect: ItemEffect{
				Type:   ItemEffectType(row.EffectType),
				Target: row.EffectTarget,
				Value:  row.EffectValue,
			},
		}
	}
	c := &Conf{configs: configs}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("item validate: %w", err)
	}
	return c, nil
}

// New 构造道具配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
