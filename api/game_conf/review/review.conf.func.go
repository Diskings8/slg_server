package review

import (
	"fmt"
)

// Validate 校验审查配置完整性（次数/任务/经验范围/等级奖励）
func (c *Conf) Validate() error {
	if len(c.levels) == 0 {
		return fmt.Errorf("review levels must not be empty")
	}
	if c.DailyChances <= 0 {
		return fmt.Errorf("daily_chances must be > 0")
	}
	if c.MaxChances < c.DailyChances {
		return fmt.Errorf("max_chances must be >= daily_chances")
	}
	if c.TasksPerReview <= 0 {
		return fmt.Errorf("tasks_per_review must be > 0")
	}
	if c.ExpPerReviewMin <= 0 || c.ExpPerReviewMax < c.ExpPerReviewMin {
		return fmt.Errorf("invalid exp_per_review range [%d,%d]", c.ExpPerReviewMin, c.ExpPerReviewMax)
	}
	for level, cfg := range c.levels {
		if level != cfg.Level {
			return fmt.Errorf("review map key %d != level %d", level, cfg.Level)
		}
		if cfg.ExpRequired <= 0 {
			return fmt.Errorf("review level %d exp_required must be > 0", level)
		}
		if len(cfg.Rewards) == 0 {
			return fmt.Errorf("review level %d rewards must not be empty", level)
		}
		for _, r := range cfg.Rewards {
			if r.ItemID <= 0 || r.Count <= 0 {
				return fmt.Errorf("review level %d invalid reward item %d x %d", level, r.ItemID, r.Count)
			}
		}
	}
	return nil
}
