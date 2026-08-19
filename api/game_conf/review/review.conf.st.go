// Package review 审查玩法配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
//
// 玩法：每天 8:00 后获得 1 次审查次数（最多累积 5 次）；消耗 1 次生成 tasks_per_review 个任务
// 并 +exp 审查经验；审查等级决定任务奖励品质；前 level_up_bonus_chances 级每升 1 级送 1 次审查次数。
// 任务奖励为道具（后续可转化为资源，转化暂未实现）。
package review

// ReviewReward 单任务奖励（道具）
type ReviewReward struct {
	ItemID int32 `json:"item_id"` // 道具配置ID（item 表）
	Count  int32 `json:"count"`   // 数量
}

// ReviewLevelConf 单等级审查配置（亦为 JSON 表行结构）
type ReviewLevelConf struct {
	Level       int32          `json:"level"`        // 审查等级
	ExpRequired int32          `json:"exp_required"` // 升到该等级所需累计经验
	Rewards     []ReviewReward `json:"rewards"`      // 该等级任务奖励（道具）
}

// Conf 审查玩法配置聚合
type Conf struct {
	DailyChances        int32 // 每天 8:00 后获得次数
	MaxChances          int32 // 最多累积次数
	TasksPerReview      int32 // 每次审查生成任务数
	ExpPerReviewMin     int32 // 每次审查最低经验
	ExpPerReviewMax     int32 // 每次审查最高经验
	SeasonDays          int32 // 赛季天数
	LevelUpBonusChances int32 // 前 N 级每升 1 级送 1 次审查次数

	levels map[int32]*ReviewLevelConf // 等级 → 配置
}

// GetConfig 按等级查询配置（未配置返回 nil）
func (c *Conf) GetConfig(level int32) *ReviewLevelConf {
	return c.levels[level]
}

// GetRewards 该等级任务奖励（道具列表）
func (c *Conf) GetRewards(level int32) []ReviewReward {
	cfg := c.levels[level]
	if cfg == nil {
		return nil
	}
	return cfg.Rewards
}

// GetExpRequired 升到该等级所需累计经验
func (c *Conf) GetExpRequired(level int32) int32 {
	cfg := c.levels[level]
	if cfg == nil {
		return 0
	}
	return cfg.ExpRequired
}
