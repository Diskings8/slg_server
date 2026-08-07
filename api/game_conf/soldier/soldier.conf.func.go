package soldier

import (
	"fmt"
	"sort"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// soldierJSON 兵力配置表 JSON 结构（磁盘格式，snake_case）
type soldierJSON struct {
	DefaultSoldierNum uint32              `json:"default_soldier_num"`
	HeroLevelCaps     []heroLevelCap      `json:"hero_level_caps"`     // 英雄等级 → 基础兵力断点
	BarrackLevelBonus []barrackLevelBonus `json:"barrack_level_bonus"` // 兵营等级 → 加成断点
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "soldier" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建兵力配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j soldierJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	// 局部构建（前向填充/前缀和），末尾一次性提交
	maxHero := maxLevel(j.HeroLevelCaps)
	maxBarrack := maxLevelBonus(j.BarrackLevelBonus)

	heroCaps := make([]uint32, maxHero+1)
	for _, r := range j.HeroLevelCaps {
		if int(r.Level) < len(heroCaps) {
			heroCaps[r.Level] = r.SoldierNum
		}
	}
	prev := uint32(0)
	for i := 1; i < len(heroCaps); i++ {
		if heroCaps[i] != 0 {
			prev = heroCaps[i]
		} else {
			heroCaps[i] = prev
		}
	}

	barrackBonus := make([]uint32, maxBarrack+1)
	for _, r := range j.BarrackLevelBonus {
		if int(r.Level) < len(barrackBonus) {
			barrackBonus[r.Level] = r.Bonus
		}
	}
	prevB := uint32(0)
	for i := 1; i < len(barrackBonus); i++ {
		if barrackBonus[i] != 0 {
			prevB = barrackBonus[i]
		} else {
			barrackBonus[i] = prevB
		}
	}

	c.DefaultSoldierNum = j.DefaultSoldierNum
	c.heroCaps = heroCaps
	c.barrackBonus = barrackBonus
	c.maxHeroLevel = int32(maxHero)
	c.maxBarrackLevel = int32(maxBarrack)
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验兵力配置完整性（默认兵力>0、断点 level 递增、数值>0）
func (c *Conf) Validate() error {
	if c.DefaultSoldierNum == 0 {
		return fmt.Errorf("default_soldier_num must be > 0")
	}
	if len(c.heroCaps) == 0 || c.maxHeroLevel == 0 {
		return fmt.Errorf("hero_level_caps must not be empty")
	}
	if c.heroCaps[1] == 0 {
		return fmt.Errorf("hero_level_caps[1] must be > 0")
	}
	if len(c.barrackBonus) == 0 {
		return fmt.Errorf("barrack_level_bonus must not be empty")
	}
	return nil
}

// maxLevel 返回断点最大英雄等级（断点 level>0）
func maxLevel(rows []heroLevelCap) int {
	m := 0
	for _, r := range rows {
		if int(r.Level) > m {
			m = int(r.Level)
		}
	}
	return m
}

// maxLevelBonus 返回断点最大兵营等级
func maxLevelBonus(rows []barrackLevelBonus) int {
	m := 0
	for _, r := range rows {
		if int(r.Level) > m {
			m = int(r.Level)
		}
	}
	return m
}

// AllLevels 所有英雄等级（升序，供测试/校验用）
func (c *Conf) AllLevels() []int32 {
	levels := make([]int32, 0, c.maxHeroLevel)
	for i := int32(1); i <= c.maxHeroLevel; i++ {
		levels = append(levels, i)
	}
	return levels
}

// AllBarrackLevels 所有兵营等级（升序）
func (c *Conf) AllBarrackLevels() []int32 {
	levels := make([]int32, 0, c.maxBarrackLevel)
	for i := int32(1); i <= c.maxBarrackLevel; i++ {
		levels = append(levels, i)
	}
	return levels
}

// SortedHeroCaps 英雄等级断点（升序，测试断言用）
func (c *Conf) SortedHeroCaps() []heroLevelCap {
	rows := make([]heroLevelCap, 0)
	for i := 1; i < len(c.heroCaps); i++ {
		if i == 1 || c.heroCaps[i] != c.heroCaps[i-1] {
			rows = append(rows, heroLevelCap{Level: int32(i), SoldierNum: c.heroCaps[i]})
		}
	}
	sort.Slice(rows, func(a, b int) bool { return rows[a].Level < rows[b].Level })
	return rows
}
