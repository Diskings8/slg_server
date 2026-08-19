package exchange

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
)

// NewFromPB 从 tabtoy 单一配置构建货币兑换配置（exchange 表逐行 → rules 索引）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	rules := make(map[pb_confs.ItemID]*RuleConfig, len(t.Exchange))
	for _, row := range t.Exchange {
		if _, dup := rules[pb_confs.ItemID(row.FromId)]; dup {
			return nil, fmt.Errorf("duplicate exchange from_id %d", row.FromId)
		}
		rules[pb_confs.ItemID(row.FromId)] = &RuleConfig{
			FromID:    pb_confs.ItemID(row.FromId),
			FromType:  pb_confs.ItemType(row.FromType),
			ToID:      pb_confs.ItemID(row.ToId),
			ToType:    pb_confs.ItemType(row.ToType),
			FromCount: row.FromCount,
			ToCount:   row.ToCount,
		}
	}
	c := &Conf{rules: rules}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("exchange validate: %w", err)
	}
	return c, nil
}

// New 构造货币兑换配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
