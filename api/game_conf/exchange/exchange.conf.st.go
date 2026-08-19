// Package exchange 货币兑换配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
//
// 货币兑换是货币流通渠道：一级货币（钻石，仅充值获得）→ 二级货币（金币，游戏内主要消耗）。
// 比例由配置驱动，一个来源货币一条规则（当前占位 1 钻石 = 10 金币）。
package exchange

import "server.slg.com/api/protocol/pb_confs"

// RuleConfig 货币兑换规则（一级→二级）
type RuleConfig struct {
	FromID    pb_confs.ItemID   `json:"from_id"`    // 来源货币配置ID（一级货币，如钻石）
	FromType  pb_confs.ItemType `json:"from_type"`  // 来源货币类型
	ToID      pb_confs.ItemID   `json:"to_id"`      // 目标货币配置ID（二级货币，如金币）
	ToType    pb_confs.ItemType `json:"to_type"`    // 目标货币类型
	FromCount int64             `json:"from_count"` // 消耗来源货币数量（每组）
	ToCount   int64             `json:"to_count"`   // 获得目标货币数量（每组）
}

// Conf 货币兑换配置聚合（按 FromID 索引，一个来源货币一条规则）
type Conf struct {
	rules map[pb_confs.ItemID]*RuleConfig
}

// GetRule 按来源货币配置ID查询兑换规则
func (c *Conf) GetRule(fromID pb_confs.ItemID) (*RuleConfig, bool) {
	rule, ok := c.rules[fromID]
	return rule, ok
}
