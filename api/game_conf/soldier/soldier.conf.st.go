// Package soldier 兵力上限配置表（英雄上阵默认兵力 + 等级/兵营加成，Go 内嵌占位数据，后续可迁 JSON）
//
// 兵力模型：英雄上阵默认给 default_soldier_num 兵；兵力上限 = 英雄等级基础 + 兵营等级累计加成。
// 断点数据（hero_level_caps / barrack_level_bonus）在 Load 时展开为稠密数组，运行期 O(1) 查询。
package soldier

// heroLevelCap 英雄等级兵力断点（稀疏，Load 时按"最大 ≤ 等级"前向填充）
type heroLevelCap struct {
	Level      int32  `json:"level"`       // 英雄等级
	SoldierNum uint32 `json:"soldier_num"` // 该等级及以上的基础兵力
}

// barrackLevelBonus 兵营等级兵力加成断点（稀疏；bonus 为该等级的累计加成）
type barrackLevelBonus struct {
	Level int32  `json:"level"` // 兵营等级
	Bonus uint32 `json:"bonus"` // 该等级的累计兵力加成
}

// Conf 兵力上限配置聚合
type Conf struct {
	DefaultSoldierNum uint32   // 上阵默认兵力
	heroCaps          []uint32 // index=英雄等级 1..MaxHeroLevel（最大 ≤ 等级的基础兵力）
	barrackBonus      []uint32 // index=兵营等级 1..MaxBarrackLevel（累计加成）
	maxHeroLevel      int32    // 英雄等级上限（heroCaps 长度）
	maxBarrackLevel   int32    // 兵营等级上限（barrackBonus 长度）

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造兵力配置（内置占位数据）
func New() *Conf {
	c := &Conf{
		DefaultSoldierNum: 100,
		maxHeroLevel:      20,
		maxBarrackLevel:   3,
	}
	// 英雄等级基础兵力：1级100，10级200，20级350（前向填充）
	c.heroCaps = make([]uint32, c.maxHeroLevel+1)
	fillHeroCaps(c.heroCaps, []heroLevelCap{
		{Level: 1, SoldierNum: 100},
		{Level: 10, SoldierNum: 200},
		{Level: 20, SoldierNum: 350},
	})
	// 兵营等级累计加成：1级+0，2级+50，3级+100（断点间前向填充）
	c.barrackBonus = make([]uint32, c.maxBarrackLevel+1)
	fillBarrackBonus(c.barrackBonus, []barrackLevelBonus{
		{Level: 1, Bonus: 0},
		{Level: 2, Bonus: 50},
		{Level: 3, Bonus: 100},
	})
	return c
}

// SoldierLimit 计算兵力上限 = 英雄等级基础 + 兵营等级累计加成
func (c *Conf) SoldierLimit(heroLevel, barrackLevel uint32) uint32 {
	base := uint32(0)
	if int32(heroLevel) <= c.maxHeroLevel {
		base = c.heroCaps[heroLevel]
	} else if c.maxHeroLevel > 0 {
		base = c.heroCaps[c.maxHeroLevel]
	}

	bonus := uint32(0)
	if barrackLevel > 0 {
		if int32(barrackLevel) <= c.maxBarrackLevel {
			bonus = c.barrackBonus[barrackLevel]
		} else if c.maxBarrackLevel > 0 {
			bonus = c.barrackBonus[c.maxBarrackLevel]
		}
	}

	return base + bonus
}

// fillHeroCaps 前向填充英雄等级基础兵力（每个断点生效到下一个断点前）
func fillHeroCaps(caps []uint32, rows []heroLevelCap) {
	// 先把断点写入对应等级
	for _, r := range rows {
		if int(r.Level) < len(caps) {
			caps[r.Level] = r.SoldierNum
		}
	}
	// 前向填充：断点之间用前一个值
	prev := uint32(0)
	for i := 1; i < len(caps); i++ {
		if caps[i] != 0 {
			prev = caps[i]
		} else {
			caps[i] = prev
		}
	}
}

// fillBarrackBonus 前向填充兵营等级累计加成（每个断点生效到下一个断点前）
func fillBarrackBonus(bonus []uint32, rows []barrackLevelBonus) {
	// 先把断点写入对应等级
	for _, r := range rows {
		if int(r.Level) < len(bonus) {
			bonus[r.Level] = r.Bonus
		}
	}
	// 前向填充：断点之间用前一个值
	prev := uint32(0)
	for i := 1; i < len(bonus); i++ {
		if bonus[i] != 0 {
			prev = bonus[i]
		} else {
			bonus[i] = prev
		}
	}
}
