package role_reviews

import (
	"errors"
	"math/rand/v2"
	"time"

	"server.slg.com/api/game_conf/review"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

// ErrNoChance 无审查次数
var ErrNoChance = errors.New("无审查次数")

// reviewDay 当前审查结算日（每天 8:00 切日；8 点前算前一天）
func reviewDay() int32 {
	now := time.Now()
	if now.Hour() < 8 {
		now = now.AddDate(0, 0, -1)
	}
	return int32(now.Year()%100)*10000 + int32(now.Month())*100 + int32(now.Day())
}

// settleDaily 结算每日审查次数（新的一天 +daily，封顶 max）
func (rrs *RoleReviews) settleDaily(conf *review.Conf) {
	if rrs.Review == nil {
		return
	}
	day := reviewDay()
	if rrs.Review.LastDate < day {
		rrs.Review.Chances += conf.DailyChances
		if rrs.Review.Chances > conf.MaxChances {
			rrs.Review.Chances = conf.MaxChances
		}
		rrs.Review.LastDate = day
	}
}

// StartReview 消耗一次审查次数生成任务：
//   - 结算每日次数
//   - 消耗 1 次
//   - +exp（随机 [Min,Max]），按经验曲线升级（前 N 级升级送 1 次审查次数）
//   - 生成 tasks_per_review 个任务（随机事件类型，奖励取当前审查等级配置）
func (rrs *RoleReviews) StartReview(conf *review.Conf) (tasks []ReviewTask, expGained int32, leveledUp bool, err error) {
	rrs.settleDaily(conf)
	if rrs.Review == nil {
		return nil, 0, false, ErrNoChance
	}
	if rrs.Review.Chances <= 0 {
		return nil, 0, false, ErrNoChance
	}
	rrs.Review.Chances--

	// 经验 + 升级
	expGained = int32(rand.IntN(int(conf.ExpPerReviewMax-conf.ExpPerReviewMin+1))) + conf.ExpPerReviewMin
	rrs.Review.Exp += expGained
	oldLevel := rrs.Review.Level
	for {
		next := conf.GetExpRequired(rrs.Review.Level + 1)
		if next > 0 && rrs.Review.Exp >= next {
			rrs.Review.Level++
		} else {
			break
		}
	}
	if rrs.Review.Level > oldLevel {
		leveledUp = true
		// 前 N 级每升 1 级送 1 次审查次数（封顶）
		if rrs.Review.Level <= conf.LevelUpBonusChances {
			rrs.Review.Chances++
			if rrs.Review.Chances > conf.MaxChances {
				rrs.Review.Chances = conf.MaxChances
			}
		}
	}

	// 生成任务（固定 2 件打怪[行军]，其余随机 采集/寻宝[点击]；奖励取当前审查等级配置）
	rrs.Pending = nil
	rewards := conf.GetRewards(rrs.Review.Level)
	taskTypes := reviewTaskTypes(conf.TasksPerReview)
	for _, typ := range taskTypes {
		rrs.nextTaskID++
		task := ReviewTask{TaskID: rrs.nextTaskID, Type: typ}
		for _, rw := range rewards {
			task.Rewards = append(task.Rewards, TaskReward{ItemID: rw.ItemID, Count: rw.Count})
		}
		rrs.Pending = append(rrs.Pending, task)
	}
	return rrs.Pending, expGained, leveledUp, nil
}

// reviewTaskTypes 生成任务类型：前 2 件固定打怪（行军），其余随机 采集/寻宝（点击）
func reviewTaskTypes(total int32) []int32 {
	types := make([]int32, total)
	for i := int32(0); i < total && i < 2; i++ {
		types[i] = int32(map_events.EventTypeMonster)
	}
	for i := int32(2); i < total; i++ {
		if rand.IntN(2) == 0 {
			types[i] = int32(map_events.EventTypeResource)
		} else {
			types[i] = int32(map_events.EventTypeTreasure)
		}
	}
	return types
}

// SelectTask 选择任务：从待处理中取出并返回（从内存移除）
func (rrs *RoleReviews) SelectTask(taskID int64) (*ReviewTask, error) {
	for i, t := range rrs.Pending {
		if t.TaskID == taskID {
			rrs.Pending = append(rrs.Pending[:i], rrs.Pending[i+1:]...)
			return &t, nil
		}
	}
	return nil, errors.New("任务不存在或已执行")
}
