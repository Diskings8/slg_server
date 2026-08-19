package resource

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建资源产量配置（resource 表逐行 → configs 索引）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	configs := make(map[int32]*ResourceConfig, len(t.Resource))
	for _, row := range t.Resource {
		if row.Level <= 0 {
			return nil, fmt.Errorf("resource level must be > 0, got %d", row.Level)
		}
		if _, dup := configs[row.Level]; dup {
			return nil, fmt.Errorf("duplicate resource level %d", row.Level)
		}
		configs[row.Level] = &ResourceConfig{
			Level:           row.Level,
			Type:            int32(row.ResType),
			Amount:          row.Amount,
			PrimaryAmount:   row.PrimaryAmount,
			SecondaryAmount: row.SecondaryAmount,
		}
	}
	c := &Conf{configs: configs}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("resource validate: %w", err)
	}
	return c, nil
}

// New 构造资源产量配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
