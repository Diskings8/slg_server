package hero

import (
	"fmt"
)

// Validate 校验英雄配置完整性（主键/数值范围/表内约束）
func (c *Conf) Validate() error {
	if c.MaxLevel == 0 {
		return fmt.Errorf("max_level must be > 0")
	}
	if uint32(len(c.ExpNeed)) != c.MaxLevel {
		return fmt.Errorf("exp_need length %d != max_level %d", len(c.ExpNeed), c.MaxLevel)
	}
	for i, need := range c.ExpNeed {
		if need == 0 {
			return fmt.Errorf("exp_need[%d] must be > 0", i)
		}
	}
	if len(c.heroes) == 0 {
		return fmt.Errorf("heroes must not be empty")
	}
	for id, hc := range c.heroes {
		if id <= 0 {
			return fmt.Errorf("conf_id must be > 0, got %d", id)
		}
		if hc.Base == (HeroAttr{}) {
			return fmt.Errorf("conf_id %d base all zero", id)
		}
		if hc.Growth == (HeroAttr{}) {
			return fmt.Errorf("conf_id %d growth all zero", id)
		}
		if hc.AttackRange == 0 {
			return fmt.Errorf("conf_id %d attack_range must be > 0", id)
		}
	}
	return nil
}
