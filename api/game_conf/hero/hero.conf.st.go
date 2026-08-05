// Package hero 英雄配置表（Go 内嵌占位数据，后续可迁 JSON）
package hero

// HeroAttr 英雄战斗属性（无战力聚合，直接使用真实属性参与战斗）
type HeroAttr struct {
	Attack       uint32 `json:"attack"`       // 攻击
	Defense      uint32 `json:"defense"`      // 防御
	Intelligence uint32 `json:"intelligence"` // 智力
	Movement     uint32 `json:"movement"`     // 移动
	Relocation   uint32 `json:"relocation"`   // 拆迁
}

// HeroConf 每英雄属性配置
type HeroConf struct {
	ConfID      int32    `json:"conf_id"`
	Base        HeroAttr `json:"base"`        // 基础属性（lv1）
	Growth      HeroAttr `json:"growth"`      // 每级成长（每升 1 级增加）
	AttackRange uint32   `json:"attack_range"` // 攻击距离（能打到"距离对方大营 ≤ D"的目标，固定值）
}

// Conf 英雄配置聚合
type Conf struct {
	MaxLevel        uint32   // 英雄等级上限
	FreePointPer10L uint32   // 每10级获得的自由属性点
	MaxStarStage    int32    // 星级上限
	StarPointPer    uint32   // 每升 1 星发放的自由属性点（星级不直接乘属性，改发点由玩家分配）
	ExpNeed         []uint32 // 逐级升级经验表（index=level-1）

	heroes  map[int32]HeroConf // 每英雄属性表
	version string             // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造英雄配置（内置占位数据）
func New() *Conf {
	c := &Conf{
		MaxLevel:        100,
		FreePointPer10L: 5,
		MaxStarStage:    5,
		StarPointPer:    5,
	}
	// 占位经验曲线：从 level 升到 level+1 需 (level)*100，后续直接替换表数据
	c.ExpNeed = make([]uint32, c.MaxLevel)
	for lv := 0; lv < int(c.MaxLevel); lv++ {
		c.ExpNeed[lv] = uint32((lv + 1) * 100)
	}

	// 占位英雄属性：英雄 1~5 按稀有度递增（后续迁配置表/JSON）
	c.heroes = map[int32]HeroConf{
		1: heroConf(1,
			HeroAttr{Attack: 100, Defense: 80, Intelligence: 60, Movement: 50, Relocation: 20},
			HeroAttr{Attack: 10, Defense: 8, Intelligence: 6, Movement: 5, Relocation: 2}, 3),
		2: heroConf(2,
			HeroAttr{Attack: 120, Defense: 90, Intelligence: 70, Movement: 55, Relocation: 25},
			HeroAttr{Attack: 12, Defense: 9, Intelligence: 7, Movement: 6, Relocation: 3}, 4),
		3: heroConf(3,
			HeroAttr{Attack: 140, Defense: 100, Intelligence: 80, Movement: 60, Relocation: 30},
			HeroAttr{Attack: 14, Defense: 10, Intelligence: 8, Movement: 6, Relocation: 3}, 3),
		4: heroConf(4,
			HeroAttr{Attack: 160, Defense: 110, Intelligence: 90, Movement: 65, Relocation: 35},
			HeroAttr{Attack: 16, Defense: 11, Intelligence: 9, Movement: 7, Relocation: 4}, 4),
		5: heroConf(5,
			HeroAttr{Attack: 180, Defense: 120, Intelligence: 100, Movement: 70, Relocation: 40},
			HeroAttr{Attack: 18, Defense: 12, Intelligence: 10, Movement: 7, Relocation: 4}, 5),
	}
	return c
}

// heroConf 便捷构造每英雄属性配置
func heroConf(confID int32, base, growth HeroAttr, attackRange uint32) HeroConf {
	return HeroConf{ConfID: confID, Base: base, Growth: growth, AttackRange: attackRange}
}

// NeedExp 从 level 升到 level+1 所需经验（读逐级表；越界返回 0=已达上限）
func (c *Conf) NeedExp(level uint32) uint32 {
	if level == 0 || level > c.MaxLevel {
		return 0
	}
	return c.ExpNeed[level-1]
}

// HeroConf 按配置ID查询英雄属性配置
func (c *Conf) HeroConf(confID int32) (HeroConf, bool) {
	hc, ok := c.heroes[confID]
	return hc, ok
}

// CalcCurVal 计算英雄等级派生的基础属性（写入 Cultivate.cur_val）。
//
//	cur_val = base + growth×(level-1)
//
// 不含星级（星级改发自由属性点，不直接乘属性）、不含培养加点（加点走 add_val_camp）。
// 由 game 侧在升级/创建时调用，battle 侧只读快照里的组件，不再依赖本配置。
func (c *Conf) CalcCurVal(confID int32, level uint32) HeroAttr {
	conf, ok := c.heroes[confID]
	if !ok {
		return HeroAttr{}
	}
	if level < 1 {
		level = 1 // 防 (level-1) 下溢；快照未填等级按 1 级
	}
	return HeroAttr{
		Attack:       conf.Base.Attack + conf.Growth.Attack*(level-1),
		Defense:      conf.Base.Defense + conf.Growth.Defense*(level-1),
		Intelligence: conf.Base.Intelligence + conf.Growth.Intelligence*(level-1),
		Movement:     conf.Base.Movement + conf.Growth.Movement*(level-1),
		Relocation:   conf.Base.Relocation + conf.Growth.Relocation*(level-1),
	}
}
