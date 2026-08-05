package item

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/util_jsons"
)

// itemJSON 道具配置表 JSON 结构（磁盘格式，snake_case）
type itemJSON struct {
	Items []itemRowJSON `json:"items"`
}

// itemRowJSON 单道具配置行
type itemRowJSON struct {
	ConfID pb_confs.ItemID `json:"conf_id"`
	Effect ItemEffect      `json:"effect"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "item" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建道具配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j itemJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	configs := make(map[pb_confs.ItemID]ItemConfig, len(j.Items))
	for _, row := range j.Items {
		if row.ConfID <= 0 {
			return fmt.Errorf("conf_id must be > 0, got %d", row.ConfID)
		}
		if _, dup := configs[row.ConfID]; dup {
			return fmt.Errorf("duplicate conf_id %d", row.ConfID)
		}
		configs[row.ConfID] = ItemConfig{ConfID: row.ConfID, Effect: row.Effect}
	}

	c.configs = configs
	c.version = table.ContentHash(data)
	return nil
}

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
