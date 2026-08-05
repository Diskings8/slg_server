package hero

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// heroJSON 英雄配置表 JSON 结构（磁盘格式，snake_case）
type heroJSON struct {
	MaxLevel        uint32        `json:"max_level"`
	FreePointPer10L uint32        `json:"free_point_per_10l"`
	MaxStarStage    int32         `json:"max_star_stage"`
	StarPointPer    uint32        `json:"star_point_per"`
	ExpNeed         []uint32      `json:"exp_need"`
	Heroes          []heroRowJSON `json:"heroes"`
}

// heroRowJSON 单英雄配置行
type heroRowJSON struct {
	ConfID      int32    `json:"conf_id"`
	Base        HeroAttr `json:"base"`
	Growth      HeroAttr `json:"growth"`
	AttackRange uint32   `json:"attack_range"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "hero" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建英雄配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j heroJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// 局部构建索引，主键重复直接报错（不做静默覆盖）
	heroes := make(map[int32]HeroConf, len(j.Heroes))
	for _, row := range j.Heroes {
		if _, dup := heroes[row.ConfID]; dup {
			return fmt.Errorf("duplicate conf_id %d", row.ConfID)
		}
		heroes[row.ConfID] = HeroConf{
			ConfID:      row.ConfID,
			Base:        row.Base,
			Growth:      row.Growth,
			AttackRange: row.AttackRange,
		}
	}

	// 末尾一次性提交，保证失败不产生半更新
	c.MaxLevel = j.MaxLevel
	c.FreePointPer10L = j.FreePointPer10L
	c.MaxStarStage = j.MaxStarStage
	c.StarPointPer = j.StarPointPer
	c.ExpNeed = j.ExpNeed
	c.heroes = heroes
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验英雄配置完整性（主键/数值范围/表内约束）
func (c *Conf) Validate() error {
	if c.MaxLevel == 0 {
		return fmt.Errorf("max_level must be > 0")
	}
	if uint32(len(c.ExpNeed)) != c.MaxLevel {
		return fmt.Errorf("exp_need length %d != max_level %d", len(c.ExpNeed), c.MaxLevel)
	}
	for i, need := range c.ExpNeed {
		if need == 0 {
			return fmt.Errorf("exp_need[%d] must be > 0", i)
		}
	}
	if len(c.heroes) == 0 {
		return fmt.Errorf("heroes must not be empty")
	}
	for id, hc := range c.heroes {
		if id <= 0 {
			return fmt.Errorf("conf_id must be > 0, got %d", id)
		}
		if hc.Base == (HeroAttr{}) {
			return fmt.Errorf("conf_id %d base all zero", id)
		}
		if hc.Growth == (HeroAttr{}) {
			return fmt.Errorf("conf_id %d growth all zero", id)
		}
		if hc.AttackRange == 0 {
			return fmt.Errorf("conf_id %d attack_range must be > 0", id)
		}
	}
	return nil
}
