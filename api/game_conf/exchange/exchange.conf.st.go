// Package exchange 货币兑换配置表（Go 内嵌占位数据，后续可迁 JSON）
//
// 货币兑换是货币流通渠道：一级货币（钻石，仅充值获得）→ 二级货币（金币，游戏内主要消耗）。
// 比例由配置驱动，一个来源货币一条规则（当前占位 1 钻石 = 10 金币）。
package exchange

import "server.slg.com/api/protocol/pb_confs"

// RuleConfig 货币兑换规则（一级→二级）
type RuleConfig struct {
	FromID    pb_confs.ItemID   // 来源货币配置ID（一级货币，如钻石）
	FromType  pb_confs.ItemType // 来源货币类型
	ToID      pb_confs.ItemID   // 目标货币配置ID（二级货币，如金币）
	ToType    pb_confs.ItemType // 目标货币类型
	FromCount int64             // 消耗来源货币数量（每组）
	ToCount   int64             // 获得目标货币数量（每组）
}

// Conf 货币兑换配置聚合（按 FromID 索引，一个来源货币一条规则）
type Conf struct {
	rules map[pb_confs.ItemID]*RuleConfig
}

// New 构造货币兑换配置（内置占位数据）
func New() *Conf {
	return &Conf{
		rules: map[pb_confs.ItemID]*RuleConfig{
			pb_confs.Currency1ConfID: {
				FromID:    pb_confs.Currency1ConfID,
				FromType:  pb_confs.ItemTypeCurrency1,
				ToID:      pb_confs.Currency2ConfID,
				ToType:    pb_confs.ItemTypeCurrency2,
				FromCount: 1,
				ToCount:   10,
			},
		},
	}
}

// GetRule 按来源货币配置ID查询兑换规则
func (c *Conf) GetRule(fromID pb_confs.ItemID) (*RuleConfig, bool) {
	rule, ok := c.rules[fromID]
	return rule, ok
}
