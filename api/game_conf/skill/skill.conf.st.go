// Package skill 技能配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
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
