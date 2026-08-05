// Package skill 技能配置表（Go 内嵌占位数据，后续可迁 JSON）
package skill

import "server.slg.com/common/common_declarations"

// SkillType 技能类型（战斗执行分类）
type SkillType int32

const (
	SkillTypeNone    SkillType = 0
	SkillTypeActive  SkillType = 1 // 主动技能（行动时先手释放）
	SkillTypePursuit SkillType = 2 // 追击技能（普攻后按概率触发）
	SkillTypePassive SkillType = 3 // 被动技能（buff，预留）
)

// TargetType 技能目标选择
type TargetType int32

const (
	TargetNone   TargetType = 0
	TargetRandom TargetType = 1 // 攻击范围内随机单个
	TargetFront  TargetType = 2 // 前排（距己方大营最近）
	TargetBase   TargetType = 3 // 大营
)

// EffectType 技能效果类型
type EffectType int32

const (
	EffectNone        EffectType = 0
	EffectPhysDamage  EffectType = 1 // 物理伤害：攻击 vs 防御，收敛公式
	EffectMagicDamage EffectType = 2 // 法术伤害：双方智力，收敛公式
	EffectRecover     EffectType = 3 // 恢复伤兵
)

// SkillConf 技能配置
type SkillConf struct {
	ConfID      int32
	MaxLevel    int32                       // 等级上限
	UseLimit    int32                       // 可装配次数上限
	UpgradeCost common_declarations.ItemUse // 单次升级消耗（ItemType 默认 0=Normal）

	// ── 战斗字段（battle 节点加载技能表执行） ──
	SkillType     SkillType  // 技能类型
	TargetType    TargetType // 目标选择
	EffectType    EffectType // 效果类型
	DamageCoeff   uint32     // 技能伤害系数（%），0=按基础攻/智算
	ConvergeCoeff uint32     // 技能收敛系数（%），0=用战斗规则全局系数
	TriggerRate   uint32     // 追击触发概率（%），仅追击技能有效
}

// SkillCollectionConf 技能收藏配置（分次收集，所需英雄卡列表）
//
// 消耗英雄卡推进收集进度，全部达标解锁对应技能。
// NeedHeroes 的 ItemID = 英雄配置ID（与 gacha 掉落/升星消耗同源）。
type SkillCollectionConf struct {
	SkillConfID int32
	NeedHeroes  []common_declarations.ItemUse // 所需英雄卡（ItemID=英雄配置ID + 所需数量）
}

// Conf 技能配置聚合
type Conf struct {
	skillList       []SkillConf
	skillByID       map[int32]SkillConf
	collectionConfs map[int32]SkillCollectionConf

	// ── 技能槽位配置 ──
	SlotDefault   int32  // index0 默认技能槽（英雄自带）
	SlotEquipMin  int32  // 可装配槽位起始
	SlotEquipMax  int32  // 可装配槽位上限
	Slot1UnlockLv uint32 // 槽位1（index1）解锁所需英雄等级
	Slot2UnlockLv uint32 // 槽位2（index2）解锁所需英雄等级
	UnequipRefund int32  // 拆卸返还比例：每升级 1 级返还 1/2 升级道具
}

// New 构造技能配置（内置占位数据）
func New() *Conf {
	c := &Conf{
		skillList: []SkillConf{
			{ConfID: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: common_declarations.ItemUse{ItemID: 2001, Count: 1},
				SkillType: SkillTypeActive, TargetType: TargetRandom, EffectType: EffectPhysDamage, DamageCoeff: 100},
			{ConfID: 102, MaxLevel: 10, UseLimit: 2, UpgradeCost: common_declarations.ItemUse{ItemID: 2001, Count: 2},
				SkillType: SkillTypePursuit, TargetType: TargetRandom, EffectType: EffectPhysDamage, DamageCoeff: 80, TriggerRate: 30},
			{ConfID: 103, MaxLevel: 5, UseLimit: 1, UpgradeCost: common_declarations.ItemUse{ItemID: 2002, Count: 1},
				SkillType: SkillTypeActive, TargetType: TargetBase, EffectType: EffectMagicDamage, DamageCoeff: 120},
		},
		collectionConfs: map[int32]SkillCollectionConf{
			101: {SkillConfID: 101, NeedHeroes: []common_declarations.ItemUse{
				{ItemID: 1, Count: 5},
				{ItemID: 2, Count: 3},
			}},
			102: {SkillConfID: 102, NeedHeroes: []common_declarations.ItemUse{
				{ItemID: 1, Count: 10},
			}},
		},
		SlotDefault:   0,
		SlotEquipMin:  1,
		SlotEquipMax:  2,
		Slot1UnlockLv: 10,
		Slot2UnlockLv: 20,
		UnequipRefund: 2,
	}
	c.skillByID = make(map[int32]SkillConf, len(c.skillList))
	for _, s := range c.skillList {
		c.skillByID[s.ConfID] = s
	}
	return c
}

// GetSkillConf 按配置ID查询技能配置
func (c *Conf) GetSkillConf(confID int32) (SkillConf, bool) {
	s, ok := c.skillByID[confID]
	return s, ok
}

// GetCollectionConf 按技能配置ID查询收藏配置
func (c *Conf) GetCollectionConf(skillConfID int32) (SkillCollectionConf, bool) {
	cc, ok := c.collectionConfs[skillConfID]
	return cc, ok
}
