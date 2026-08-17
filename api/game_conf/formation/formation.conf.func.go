package formation

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// formationJSON 编队配置表 JSON 结构（磁盘格式，snake_case）
type formationJSON struct {
	MaxSlots int `json:"max_slots"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "formation" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建编队配置（覆盖占位）。
func (c *Conf) Load(data []byte) error {
	var j formationJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	c.MaxSlots = j.MaxSlots
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验编队配置完整性。
func (c *Conf) Validate() error {
	if c.MaxSlots < 1 {
		return fmt.Errorf("max_slots must be >= 1, got %d", c.MaxSlots)
	}
	return nil
}
