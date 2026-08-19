package resource

import (
	"fmt"
)

// Validate 校验资源产量配置完整性（等级唯一/类型合法/产量为正）
func (c *Conf) Validate() error {
	if len(c.configs) == 0 {
		return fmt.Errorf("resource configs must not be empty")
	}
	for level, cfg := range c.configs {
		if level != cfg.Level {
			return fmt.Errorf("resource map key %d != level %d", level, cfg.Level)
		}
		switch ResourceType(cfg.Type) {
		case ResourceTypeMixed, ResourceTypeSingle:
			if cfg.Amount <= 0 {
				return fmt.Errorf("resource level %d amount must be > 0, got %d", level, cfg.Amount)
			}
		case ResourceTypeDual:
			if cfg.PrimaryAmount <= 0 {
				return fmt.Errorf("resource level %d primary_amount must be > 0, got %d", level, cfg.PrimaryAmount)
			}
			if cfg.SecondaryAmount <= 0 {
				return fmt.Errorf("resource level %d secondary_amount must be > 0, got %d", level, cfg.SecondaryAmount)
			}
		default:
			return fmt.Errorf("resource level %d unknown type %d", level, cfg.Type)
		}
	}
	return nil
}
