// Package gacha 抽卡系统配置表（Go 内嵌占位数据，后续可迁 JSON）
//
// 抽卡是英雄卡来源：消耗抽卡券/金币，按权重随机掉落英雄卡或道具。
// 英雄 ID 当前为占位 int32（hero_conf 定义表未建，见 LDL_COMPARISON 主要缺失），
// 后续接入配置表后只改本文件占位数据，不影响逻辑层。
package gacha

import "sort"

// DropItemConfig 掉落条目：命中后产出英雄卡或道具
type DropItemConfig struct {
	RewardConfID   int32 // 英雄配置ID（IsHero=true）或道具配置ID（IsHero=false）
	IsHero         bool  // true=英雄卡；false=道具
	Count          int32 // 产出数量（英雄卡恒 1）
	Weight         int32 // 组内权重
	GuaranteeReset bool  // 命中后保底计数归零（保底组的奖励项置 true）
}

// DropGroupConfig 掉落组（稀有度档位）
type DropGroupConfig struct {
	GroupID int32            // 组ID（档位）
	Weight  int32            // 非保底抽取时的组权重
	Items   []DropItemConfig // 组内条目（按 Weight 随机一条）
}

// RecruitPoolConfig 抽卡池配置
type RecruitPoolConfig struct {
	PoolID int32
	Name   string

	TicketConfID int32 // 抽卡券道具配置ID（0=无券模式）
	SingleTicket int64 // 单抽消耗券数
	SingleGold   int64 // 单抽消耗金币数（二级货币）
	TenTicket    int64 // 十连消耗券数
	TenGold      int64 // 十连消耗金币数

	FreeDaily bool // 是否开启免费（每天 0/12 点窗口各 1 次）
	HalfPrice bool // 是否开启半价（每天 0/12 点窗口各 1 次，金币减半）

	DropGroups       []DropGroupConfig
	GuaranteeTimes   int32 // 累抽 N 次必出高稀有度
	GuaranteeGroupID int32 // 保底命中时走的掉落组
	FirstDropGroupID int32 // 首抽保底组（0=无）

	WishHeros []int32 // 心愿可选英雄配置ID集合
	WishTimes int32   // 心愿进度阈值
}

// Conf 抽卡配置聚合
type Conf struct {
	pools map[int32]*RecruitPoolConfig
}

// New 构造抽卡配置（内置占位数据）
func New() *Conf {
	return &Conf{
		pools: map[int32]*RecruitPoolConfig{
			1001: newPool(1001, "新手池", true, true,
				1, 100, 10, 900,
				10, 3, 2,
				[]int32{2, 3}, 20,
				70, 25, 5),
			1002: newPool(1002, "英雄池", false, true,
				1, 200, 10, 1800,
				20, 3, 0,
				[]int32{1, 2, 3, 4, 5}, 50,
				60, 30, 10),
		},
	}
}

// newPool 便捷构造抽卡池，普通/稀有/史诗三档组权重入参
func newPool(poolID int32, name string, freeDaily, halfPrice bool,
	singleTicket, singleGold, tenTicket, tenGold int64,
	guaranteeTimes, guaranteeGroupID, firstDropGroupID int32,
	wishHeros []int32, wishTimes int32,
	commonW, rareW, epicW int32) *RecruitPoolConfig {

	return &RecruitPoolConfig{
		PoolID:           poolID,
		Name:             name,
		TicketConfID:     2004,
		SingleTicket:     singleTicket,
		SingleGold:       singleGold,
		TenTicket:        tenTicket,
		TenGold:          tenGold,
		FreeDaily:        freeDaily,
		HalfPrice:        halfPrice,
		DropGroups:       buildDropGroups(commonW, rareW, epicW),
		GuaranteeTimes:   guaranteeTimes,
		GuaranteeGroupID: guaranteeGroupID,
		FirstDropGroupID: firstDropGroupID,
		WishHeros:        wishHeros,
		WishTimes:        wishTimes,
	}
}

// buildDropGroups 三档稀有度掉落组（普通/稀有/史诗）
//
//   - 普通：英雄1 + 道具（2001 经验书×5 / 2002 金币包×1）
//   - 稀有：英雄2/英雄3
//   - 史诗：英雄4/英雄5，命中即保底重置（GuaranteeReset=true）
func buildDropGroups(commonW, rareW, epicW int32) []DropGroupConfig {
	return []DropGroupConfig{
		{
			GroupID: 1,
			Weight:  commonW,
			Items: []DropItemConfig{
				{RewardConfID: 1, IsHero: true, Count: 1, Weight: 40},
				{RewardConfID: 2001, IsHero: false, Count: 5, Weight: 30},
				{RewardConfID: 2002, IsHero: false, Count: 1, Weight: 30},
			},
		},
		{
			GroupID: 2,
			Weight:  rareW,
			Items: []DropItemConfig{
				{RewardConfID: 2, IsHero: true, Count: 1, Weight: 60},
				{RewardConfID: 3, IsHero: true, Count: 1, Weight: 40},
			},
		},
		{
			GroupID: 3,
			Weight:  epicW,
			Items: []DropItemConfig{
				{RewardConfID: 4, IsHero: true, Count: 1, Weight: 50, GuaranteeReset: true},
				{RewardConfID: 5, IsHero: true, Count: 1, Weight: 50, GuaranteeReset: true},
			},
		},
	}
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
