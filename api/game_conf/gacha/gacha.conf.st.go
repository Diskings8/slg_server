// Package gacha 抽卡系统配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
//
// 抽卡是英雄卡来源：消耗抽卡券/金币，按权重随机掉落英雄卡或道具。
// 三张表拼装：gacha_pool（池定义） + gacha_drop_group（稀有度档位） + gacha_drop_item（组内条目）。
package gacha

import "sort"

// DropItemConfig 掉落条目：命中后产出英雄卡或道具
type DropItemConfig struct {
	RewardConfID   int32 `json:"reward_conf_id"` // 英雄配置ID（IsHero=true）或道具配置ID（IsHero=false）
	IsHero         bool  `json:"is_hero"`        // true=英雄卡；false=道具
	Count          int32 `json:"count"`          // 产出数量（英雄卡恒 1）
	Weight         int32 `json:"weight"`         // 组内权重
	GuaranteeReset bool  `json:"guarantee_reset"` // 命中后保底计数归零（保底组的奖励项置 true）
}

// DropGroupConfig 掉落组（稀有度档位）
type DropGroupConfig struct {
	GroupID int32            `json:"group_id"` // 组ID（档位）
	Weight  int32            `json:"weight"`   // 非保底抽取时的组权重
	Items   []DropItemConfig `json:"items"`    // 组内条目（按 Weight 随机一条）
}

// RecruitPoolConfig 抽卡池配置（亦为 JSON 表行结构）
type RecruitPoolConfig struct {
	PoolID       int32  `json:"pool_id"`
	Name         string `json:"name"`
	TicketConfID int32  `json:"ticket_conf_id"` // 抽卡券道具配置ID（0=无券模式）
	SingleTicket int64  `json:"single_ticket"`  // 单抽消耗券数
	SingleGold   int64  `json:"single_gold"`    // 单抽消耗金币数（二级货币）
	TenTicket    int64  `json:"ten_ticket"`     // 十连消耗券数
	TenGold      int64  `json:"ten_gold"`       // 十连消耗金币数

	FreeDaily bool `json:"free_daily"` // 是否开启免费（每天 0/12 点窗口各 1 次）
	HalfPrice bool `json:"half_price"` // 是否开启半价（每天 0/12 点窗口各 1 次，金币减半）

	DropGroups       []DropGroupConfig `json:"drop_groups"`
	GuaranteeTimes   int32             `json:"guarantee_times"`   // 累抽 N 次必出高稀有度
	GuaranteeGroupID int32             `json:"guarantee_group_id"` // 保底命中时走的掉落组
	FirstDropGroupID int32             `json:"first_drop_group_id"` // 首抽保底组（0=无）

	WishHeros []int32 `json:"wish_heros"` // 心愿可选英雄配置ID集合
	WishTimes int32   `json:"wish_times"` // 心愿进度阈值
}

// Conf 抽卡配置聚合
type Conf struct {
	pools map[int32]*RecruitPoolConfig
}

// GetPool 按池ID查询抽卡池配置
func (c *Conf) GetPool(poolID int32) (*RecruitPoolConfig, bool) {
	pool, ok := c.pools[poolID]
	return pool, ok
}

// AllPoolIDs 所有抽卡池ID（升序，保证输出确定性）
func (c *Conf) AllPoolIDs() []int32 {
	ids := make([]int32, 0, len(c.pools))
	for id := range c.pools {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
