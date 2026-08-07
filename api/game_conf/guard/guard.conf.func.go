package guard

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// guardJSON 守军配置表 JSON 结构（多表嵌套：configs → slots）。
// 复用 GuardConfig/GuardSlot 作为表行（已带 json tag）。
type guardJSON struct {
	Configs         []*GuardConfig `json:"configs"`
	MaxDevelopLevel int32          `json:"max_develop_level"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "guard" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建守军配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j guardJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	configs := make(map[int32]*GuardConfig, len(j.Configs))
	for _, g := range j.Configs {
		if g.Level < 0 {
			return fmt.Errorf("guard level must be >= 0, got %d", g.Level)
		}
		if _, dup := configs[g.Level]; dup {
			return fmt.Errorf("duplicate guard level %d", g.Level)
		}
		configs[g.Level] = g
	}

	c.configs = configs
	c.MaxDevelopLevel = j.MaxDevelopLevel
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验守军配置完整性（等级非负/唯一/槽位非空/上限一致）
func (c *Conf) Validate() error {
	if len(c.configs) == 0 {
		return fmt.Errorf("guard configs must not be empty")
	}
	if c.MaxDevelopLevel < 0 {
		return fmt.Errorf("max_develop_level must be >= 0, got %d", c.MaxDevelopLevel)
	}
	for level, g := range c.configs {
		if level != g.Level {
			return fmt.Errorf("guard map key %d != level %d", level, g.Level)
		}
		if len(g.Slots) == 0 {
			return fmt.Errorf("guard level %d slots must not be empty", level)
		}
		for _, s := range g.Slots {
			if s.HeroConfID <= 0 {
				return fmt.Errorf("guard level %d hero_conf_id must be > 0, got %d", level, s.HeroConfID)
			}
			if s.SoldierNum <= 0 {
				return fmt.Errorf("guard level %d soldier_num must be > 0, got %d", level, s.SoldierNum)
			}
		}
	}
	return nil
}
