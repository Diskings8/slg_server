package exchange

import (
	"fmt"

	"server.slg.com/api/protocol/pb_confs"
)

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
