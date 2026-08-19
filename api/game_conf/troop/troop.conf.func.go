package troop

import (
	"fmt"
)

// Validate 校验兵种配置完整性。
func (c *Conf) Validate() error {
	if c.TransformLevel == 0 {
		return fmt.Errorf("transform_level must be > 0")
	}
	if c.DefaultTroopID <= 0 {
		return fmt.Errorf("default_troop_id must be > 0, got %d", c.DefaultTroopID)
	}
	if c.UnlockItemConf <= 0 {
		return fmt.Errorf("unlock_item_conf must be > 0, got %d", c.UnlockItemConf)
	}
	if c.DefaultTroopID == c.UnlockItemConf {
		return fmt.Errorf("default_troop_id %d == unlock_item_conf %d", c.DefaultTroopID, c.UnlockItemConf)
	}
	return nil
}
