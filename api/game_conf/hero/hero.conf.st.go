// Package hero 英雄配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
package hero

import "server.slg.com/common/common_declarations"

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
	AwakenLevel     uint32   // 觉醒等级门槛（达级后可觉醒，解锁第三技能槽）
	AwakenCost      []common_declarations.ItemUse // 觉醒消耗（资源混搭，走 ItemChange）

	heroes  map[int32]HeroConf // 每英雄属性表
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
