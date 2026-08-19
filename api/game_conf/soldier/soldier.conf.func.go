package soldier

import (
	"fmt"
	"sort"
)

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
