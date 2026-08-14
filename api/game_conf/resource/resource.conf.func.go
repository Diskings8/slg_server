package resource

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// resourceJSON 资源产量配置表 JSON 结构
type resourceJSON struct {
	Levels []*ResourceConfig `json:"levels"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "resource" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建资源产量配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j resourceJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	configs := make(map[int32]*ResourceConfig, len(j.Levels))
	for _, g := range j.Levels {
		if g.Level <= 0 {
			return fmt.Errorf("resource level must be > 0, got %d", g.Level)
		}
		if _, dup := configs[g.Level]; dup {
			return fmt.Errorf("duplicate resource level %d", g.Level)
		}
		configs[g.Level] = g
	}

	c.configs = configs
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验资源产量配置完整性（等级唯一/类型合法/产量为正）
func (c *Conf) Validate() error {
	if len(c.configs) == 0 {
		return fmt.Errorf("resource configs must not be empty")
	}
	for level, cfg := range c.configs {
		if level != cfg.Level {
			return fmt.Errorf("resource map key %d != level %d", level, cfg.Level)
		}
		switch ResourceType(cfg.Type) {
		case ResourceTypeMixed, ResourceTypeSingle:
			if cfg.Amount <= 0 {
				return fmt.Errorf("resource level %d amount must be > 0, got %d", level, cfg.Amount)
			}
		case ResourceTypeDual:
			if cfg.PrimaryAmount <= 0 {
				return fmt.Errorf("resource level %d primary_amount must be > 0, got %d", level, cfg.PrimaryAmount)
			}
			if cfg.SecondaryAmount <= 0 {
				return fmt.Errorf("resource level %d secondary_amount must be > 0, got %d", level, cfg.SecondaryAmount)
			}
		default:
			return fmt.Errorf("resource level %d unknown type %d", level, cfg.Type)
		}
	}
	return nil
}
