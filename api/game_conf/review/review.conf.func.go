package review

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// reviewJSON 审查配置表 JSON 结构
type reviewJSON struct {
	DailyChances        int32              `json:"daily_chances"`
	MaxChances          int32              `json:"max_chances"`
	TasksPerReview      int32              `json:"tasks_per_review"`
	ExpPerReviewMin     int32              `json:"exp_per_review_min"`
	ExpPerReviewMax     int32              `json:"exp_per_review_max"`
	SeasonDays          int32              `json:"season_days"`
	LevelUpBonusChances int32              `json:"level_up_bonus_chances"`
	Levels              []*ReviewLevelConf `json:"levels"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "review" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建审查配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j reviewJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	levels := make(map[int32]*ReviewLevelConf, len(j.Levels))
	for _, g := range j.Levels {
		if g.Level <= 0 {
			return fmt.Errorf("review level must be > 0, got %d", g.Level)
		}
		if _, dup := levels[g.Level]; dup {
			return fmt.Errorf("duplicate review level %d", g.Level)
		}
		levels[g.Level] = g
	}

	c.DailyChances = j.DailyChances
	c.MaxChances = j.MaxChances
	c.TasksPerReview = j.TasksPerReview
	c.ExpPerReviewMin = j.ExpPerReviewMin
	c.ExpPerReviewMax = j.ExpPerReviewMax
	c.SeasonDays = j.SeasonDays
	c.LevelUpBonusChances = j.LevelUpBonusChances
	c.levels = levels
	c.version = table.ContentHash(data)
	return nil
}

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
