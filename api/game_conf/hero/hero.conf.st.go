// Package hero 英雄配置表（Go 内嵌占位数据，后续可迁 JSON）
package hero

// Conf 英雄配置聚合
type Conf struct {
	MaxLevel        uint32   // 英雄等级上限
	FreePointPer10L uint32   // 每10级获得的自由属性点
	MaxStarStage    int32    // 星级上限
	ExpNeed         []uint32 // 逐级升级经验表（index=level-1，可替换数据，后续迁 JSON）
}

// New 构造英雄配置（内置占位数据）
func New() *Conf {
	c := &Conf{
		MaxLevel:        100,
		FreePointPer10L: 5,
		MaxStarStage:    5,
	}
	// 占位经验曲线：从 level 升到 level+1 需 (level)*100，后续直接替换表数据
	c.ExpNeed = make([]uint32, c.MaxLevel)
	for lv := 0; lv < int(c.MaxLevel); lv++ {
		c.ExpNeed[lv] = uint32((lv + 1) * 100)
	}
	return c
}

// NeedExp 从 level 升到 level+1 所需经验（读逐级表；越界返回 0=已达上限）
func (c *Conf) NeedExp(level uint32) uint32 {
	if level == 0 || level > c.MaxLevel {
		return 0
	}
	return c.ExpNeed[level-1]
}
