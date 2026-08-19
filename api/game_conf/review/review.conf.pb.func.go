package review

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建审查配置（review 单行 + review_level 等级奖励）。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Review) == 0 {
		return nil, fmt.Errorf("review table empty")
	}
	r := t.Review[0]
	c := &Conf{
		DailyChances:        r.DailyChances,
		MaxChances:          r.MaxChances,
		TasksPerReview:      r.TasksPerReview,
		ExpPerReviewMin:     r.ExpPerReviewMin,
		ExpPerReviewMax:     r.ExpPerReviewMax,
		SeasonDays:          r.SeasonDays,
		LevelUpBonusChances: r.LevelUpBonusChances,
		levels:              make(map[int32]*ReviewLevelConf, len(t.ReviewLevel)),
	}
	for _, lv := range t.ReviewLevel {
		if lv.Level <= 0 {
			return nil, fmt.Errorf("review level must be > 0, got %d", lv.Level)
		}
		if _, dup := c.levels[lv.Level]; dup {
			return nil, fmt.Errorf("duplicate review level %d", lv.Level)
		}
		rewards := make([]ReviewReward, 0, len(lv.Rewards))
		for _, rw := range lv.Rewards {
			rewards = append(rewards, ReviewReward{ItemID: int32(rw.ItemId), Count: int32(rw.Count)})
		}
		c.levels[lv.Level] = &ReviewLevelConf{Level: lv.Level, ExpRequired: lv.ExpRequired, Rewards: rewards}
	}

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("review validate: %w", err)
	}
	return c, nil
}

// New 构造审查配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
