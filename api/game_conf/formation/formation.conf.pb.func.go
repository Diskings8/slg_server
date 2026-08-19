package formation

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建编队配置（formation 表单行）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Formation) == 0 {
		return nil, fmt.Errorf("formation table empty")
	}
	c := &Conf{MaxSlots: int(t.Formation[0].MaxSlots)}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("formation validate: %w", err)
	}
	return c, nil
}

// New 构造编队配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
