package formation

import (
	"fmt"
)

// Validate 校验编队配置完整性。
func (c *Conf) Validate() error {
	if c.MaxSlots < 1 {
		return fmt.Errorf("max_slots must be >= 1, got %d", c.MaxSlots)
	}
	return nil
}
