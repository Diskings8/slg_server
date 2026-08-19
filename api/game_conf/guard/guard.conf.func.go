package guard

import (
	"fmt"
)

// Validate 校验守军配置完整性（等级非负/唯一/槽位非空/上限一致）
func (c *Conf) Validate() error {
	if len(c.configs) == 0 {
		return fmt.Errorf("guard configs must not be empty")
	}
	if c.MaxDevelopLevel < 0 {
		return fmt.Errorf("max_develop_level must be >= 0, got %d", c.MaxDevelopLevel)
	}
	for level, g := range c.configs {
		if level != g.Level {
			return fmt.Errorf("guard map key %d != level %d", level, g.Level)
		}
		if len(g.Slots) == 0 {
			return fmt.Errorf("guard level %d slots must not be empty", level)
		}
		for _, s := range g.Slots {
			if s.HeroConfID <= 0 {
				return fmt.Errorf("guard level %d hero_conf_id must be > 0, got %d", level, s.HeroConfID)
			}
			if s.SoldierNum <= 0 {
				return fmt.Errorf("guard level %d soldier_num must be > 0, got %d", level, s.SoldierNum)
			}
		}
	}
	return nil
}
