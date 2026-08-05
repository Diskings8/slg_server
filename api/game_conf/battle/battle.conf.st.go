// Package battle 战斗规则配置（battle 节点加载，战斗框架规则）
//
// 战斗配置 vs 通用配置拆分：本包 + skill 为"战斗配置"（battle 节点加载）；
// 英雄属性表/道具/建筑等为"通用配置"（game 加载，属性走 proto 给 battle）。
package battle

// 站位几何（站位 [0 1 2 2 1 0]：攻大营/攻1号/攻2号/守2号/守1号/守大营）由引擎侧
// stanceIndex/stanceDistance 计算（攻方槽位 0,1,2 → 索引 0,1,2；守方 → 5,4,3）。

// Conf 战斗规则配置（占位数据，后续迁 JSON/配置表）
type Conf struct {
	Rounds             uint32 // 战斗回合数（默认 8）
	InjuryRateStart    uint32 // 第1回合受伤比例（%）：伤害中转为伤兵的比例
	InjuryRateDecay    uint32 // 每回合受伤比例递减（%）
	SettlementDeadRate uint32 // 结算阶段：每回合当前伤兵转死亡兵比例（%）
	PhysConverge       uint32 // 物理伤害收敛系数（%）：攻击²/(攻击 + 防御×系数)
	MagicConverge      uint32 // 法术伤害收敛系数（%）：智力²/(智力 + 智力×系数)
	BattleExpCoeff     uint32 // 战斗经验系数：每场总经验 = 敌方平均等级 × 击杀敌兵 × 系数
}

// New 构造战斗规则配置（内置占位数据）
func New() *Conf {
	return &Conf{
		Rounds:             8,
		InjuryRateStart:    85,
		InjuryRateDecay:    10,
		SettlementDeadRate: 10,
		PhysConverge:       100, // 1.0
		MagicConverge:      100, // 1.0
		BattleExpCoeff:     5,
	}
}

// InjuryRate 第 round 回合的受伤比例（%），随回合衰减，最低 0
func (c *Conf) InjuryRate(round uint32) uint32 {
	r := c.InjuryRateStart
	if round > 1 {
		decay := (round - 1) * c.InjuryRateDecay
		if decay >= r {
			return 0
		}
		r -= decay
	}
	return r
}

// PhysCoeff 物理收敛系数（百分数转小数）
func (c *Conf) PhysCoeff() float64 {
	return float64(c.PhysConverge) / 100
}

// MagicCoeff 法术收敛系数（百分数转小数）
func (c *Conf) MagicCoeff() float64 {
	return float64(c.MagicConverge) / 100
}
