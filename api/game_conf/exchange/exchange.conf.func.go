package exchange

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/util_jsons"
)

// exchangeJSON 货币兑换配置表 JSON 结构（磁盘格式，snake_case）
type exchangeJSON struct {
	Rules []ruleRowJSON `json:"rules"`
}

// ruleRowJSON 单条兑换规则行
type ruleRowJSON struct {
	FromID    pb_confs.ItemID   `json:"from_id"`
	FromType  pb_confs.ItemType `json:"from_type"`
	ToID      pb_confs.ItemID   `json:"to_id"`
	ToType    pb_confs.ItemType `json:"to_type"`
	FromCount int64             `json:"from_count"`
	ToCount   int64             `json:"to_count"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "exchange" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建货币兑换配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j exchangeJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	rules := make(map[pb_confs.ItemID]*RuleConfig, len(j.Rules))
	for i := range j.Rules {
		row := &j.Rules[i]
		if _, dup := rules[row.FromID]; dup {
			return fmt.Errorf("duplicate from_id %d", row.FromID)
		}
		rules[row.FromID] = &RuleConfig{
			FromID:    row.FromID,
			FromType:  row.FromType,
			ToID:      row.ToID,
			ToType:    row.ToType,
			FromCount: row.FromCount,
			ToCount:   row.ToCount,
		}
	}

	c.rules = rules
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验货币兑换配置完整性（主键唯一/货币类型/比例/一级→二级语义）
func (c *Conf) Validate() error {
	if len(c.rules) == 0 {
		return fmt.Errorf("rules must not be empty")
	}
	for fromID, rule := range c.rules {
		if fromID <= 0 || rule.ToID <= 0 {
			return fmt.Errorf("from_id/to_id must be > 0, got %d/%d", fromID, rule.ToID)
		}
		if fromID == rule.ToID {
			return fmt.Errorf("from_id %d == to_id %d", fromID, rule.ToID)
		}
		if rule.FromCount <= 0 || rule.ToCount <= 0 {
			return fmt.Errorf("rule %d from_count/to_count must be > 0", fromID)
		}
		if rule.FromType != pb_confs.ItemTypeCurrency1 && rule.FromType != pb_confs.ItemTypeCurrency2 {
			return fmt.Errorf("rule %d invalid from_type %d", fromID, rule.FromType)
		}
		if rule.ToType != pb_confs.ItemTypeCurrency1 && rule.ToType != pb_confs.ItemTypeCurrency2 {
			return fmt.Errorf("rule %d invalid to_type %d", fromID, rule.ToType)
		}
		// 语义：一级货币（钻石）只允许兑换到二级货币（金币）
		if fromID == pb_confs.Currency1ConfID && rule.ToID != pb_confs.Currency2ConfID {
			return fmt.Errorf("rule %d: currency1 must exchange to currency2, got %d", fromID, rule.ToID)
		}
	}
	return nil
}
