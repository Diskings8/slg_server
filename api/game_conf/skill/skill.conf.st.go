// Package skill 技能配置表（Go 内嵌占位数据，后续可迁 JSON）
package skill

import "server.slg.com/common/common_declarations"

// SkillConf 技能配置
type SkillConf struct {
	ConfID      int32
	MaxLevel    int32                       // 等级上限
	UseLimit    int32                       // 可装配次数上限
	UpgradeCost common_declarations.ItemUse // 单次升级消耗（ItemType 默认 0=Normal）
}

// SkillCollectionConf 技能收藏配置（分次收集，所需道具列表）
type SkillCollectionConf struct {
	SkillConfID int32
	NeedItems   []common_declarations.ItemUse // 所需道具（ItemID + 所需数量）
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
			{ConfID: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: common_declarations.ItemUse{ItemID: 2001, Count: 1}},
			{ConfID: 102, MaxLevel: 10, UseLimit: 2, UpgradeCost: common_declarations.ItemUse{ItemID: 2001, Count: 2}},
			{ConfID: 103, MaxLevel: 5, UseLimit: 1, UpgradeCost: common_declarations.ItemUse{ItemID: 2002, Count: 1}},
		},
		collectionConfs: map[int32]SkillCollectionConf{
			101: {SkillConfID: 101, NeedItems: []common_declarations.ItemUse{
				{ItemID: 2001, Count: 5},
				{ItemID: 2002, Count: 3},
			}},
			102: {SkillConfID: 102, NeedItems: []common_declarations.ItemUse{
				{ItemID: 2001, Count: 10},
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
