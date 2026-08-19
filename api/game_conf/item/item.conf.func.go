package item

import (
	"fmt"

	"server.slg.com/api/protocol/pb_confs"
)

// Validate 校验道具配置完整性（主键唯一/效果枚举/效果字段约束/表内引用）
func (c *Conf) Validate() error {
	if len(c.configs) == 0 {
		return fmt.Errorf("items must not be empty")
	}
	for id, ic := range c.configs {
		switch ic.Effect.Type {
		case EffectNone:
			if ic.Effect.Target != 0 || ic.Effect.Value != 0 {
				return fmt.Errorf("item %d EffectNone must have zero target/value", id)
			}
		case EffectAddHeroExp:
			if ic.Effect.Value <= 0 {
				return fmt.Errorf("item %d AddHeroExp value must be > 0", id)
			}
		case EffectAddCurrency:
			t := ic.Effect.Target
			if t != int32(pb_confs.Currency1ConfID) && t != int32(pb_confs.Currency2ConfID) {
				return fmt.Errorf("item %d AddCurrency target %d invalid (want currency conf id)", id, t)
			}
		case EffectAddItem:
			if ic.Effect.Value <= 0 {
				return fmt.Errorf("item %d AddItem value must be > 0", id)
			}
			if _, ok := c.configs[pb_confs.ItemID(ic.Effect.Target)]; !ok {
				return fmt.Errorf("item %d AddItem target %d not found in items", id, ic.Effect.Target)
			}
		default:
			return fmt.Errorf("item %d invalid effect type %d", id, ic.Effect.Type)
		}
	}
	return nil
}
