package guard

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建守军配置（guard 单行 + guard_config/guard_slot 拼装槽位）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Guard) == 0 {
		return nil, fmt.Errorf("guard table empty")
	}

	// 等级 → 守军配置（槽位稍后填充）
	configs := make(map[int32]*GuardConfig, len(t.GuardConfig))
	for _, g := range t.GuardConfig {
		if g.Level < 0 {
			return nil, fmt.Errorf("guard level must be >= 0, got %d", g.Level)
		}
		if _, dup := configs[g.Level]; dup {
			return nil, fmt.Errorf("duplicate guard level %d", g.Level)
		}
		configs[g.Level] = &GuardConfig{Level: g.Level}
	}
	for _, s := range t.GuardSlot {
		cfg := configs[s.Level]
		if cfg == nil {
			return nil, fmt.Errorf("guard_slot level %d not found in guard_config", s.Level)
		}
		cfg.Slots = append(cfg.Slots, GuardSlot{HeroConfID: s.HeroConfId, SoldierNum: s.SoldierNum})
	}

	c := &Conf{configs: configs, MaxDevelopLevel: t.Guard[0].MaxDevelopLevel}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("guard validate: %w", err)
	}
	return c, nil
}

// New 构造守军配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
