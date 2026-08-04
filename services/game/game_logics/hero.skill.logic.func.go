package game_logics

import (
	"errors"

	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// ── 技能槽位配置占位常量 ──────────────────────────────────────────
// TODO: 接入配置表（api/game_conf/*.json）
const (
	skillDefaultSlot    int32 = 0  // index0 默认技能槽（英雄自带）
	skillEquipSlotMin   int32 = 1  // 可装配槽位起始
	skillEquipSlotMax   int32 = 2  // 可装配槽位上限
	skillSlot1UnlockLv  uint32 = 10 // 槽位1（index1）解锁所需英雄等级
	skillSlot2UnlockLv  uint32 = 20 // 槽位2（index2）解锁所需英雄等级
	skillUnequipRefund  int32 = 2  // 拆卸返还比例：每升级 1 级返还 1/2 升级道具
)

// SkillConf 技能配置
type SkillConf struct {
	ConfID      int32
	MaxLevel    int32                       // 等级上限
	UseLimit    int32                       // 可装配次数上限
	UpgradeCost common_declarations.ItemUse // 单次升级消耗（ItemType 默认 0=Normal）
}

// skillConfList 技能配置占位常量
var skillConfList = []SkillConf{
	{ConfID: 101, MaxLevel: 10, UseLimit: 3, UpgradeCost: common_declarations.ItemUse{ItemID: pb_confs.ItemID(2001), Count: 1}},
	{ConfID: 102, MaxLevel: 10, UseLimit: 2, UpgradeCost: common_declarations.ItemUse{ItemID: pb_confs.ItemID(2001), Count: 2}},
	{ConfID: 103, MaxLevel: 5, UseLimit: 1, UpgradeCost: common_declarations.ItemUse{ItemID: pb_confs.ItemID(2002), Count: 1}},
}

var skillConfByID = func() map[int32]SkillConf {
	m := make(map[int32]SkillConf, len(skillConfList))
	for _, c := range skillConfList {
		m[c.ConfID] = c
	}
	return m
}()

var (
	ErrSkillConfNotFound   = errors.New("skill conf not found")
	ErrSkillSlotInvalid    = errors.New("skill slot invalid")
	ErrSkillSlotLocked     = errors.New("skill slot locked")
	ErrSkillSlotOccupied   = errors.New("skill slot occupied")
	ErrSkillSlotEmpty      = errors.New("skill slot empty")
	ErrSkillNotOwned       = errors.New("skill not owned")
	ErrSkillEquippedOther  = errors.New("skill equipped by other hero")
	ErrSkillUseLimitExceed = errors.New("skill use count limit exceeded")
	ErrSkillMaxLevel       = errors.New("skill already max level")
)

// heroSkillSlotUnlocked 英雄技能槽位可用性判断（只做条件判断）
//
//   - index0：默认技能槽，始终可用
//   - index1：英雄等级 ≥ 阈值
//   - index2：英雄等级 ≥ 阈值 且 已觉醒
func heroSkillSlotUnlocked(hero *role_heroes.RoleHero, slot int32) bool {
	switch slot {
	case 1:
		return hero.GetLevel() >= skillSlot1UnlockLv
	case 2:
		return hero.GetLevel() >= skillSlot2UnlockLv && hero.GetIsAwakened()
	}
	return slot == skillDefaultSlot
}

// HeroEquipSkill 装配技能：将角色技能库技能放入英雄技能槽
//
// 前置：槽位有效/已解锁/为空、技能库存在该技能、未装配他处、装配次数未用尽。
// 装配后技能库记录被装配英雄 + 装配次数 +1；槽位技能初始等级 = 技能库等级（升级不同步）。
func HeroEquipSkill(role *game_roles.Role, hero *role_heroes.RoleHero, slot, skillConfID int32) ([]*pb_skill.Skill, error) {
	if slot < skillEquipSlotMin || slot > skillEquipSlotMax {
		return nil, ErrSkillSlotInvalid
	}
	if !heroSkillSlotUnlocked(hero, slot) {
		return nil, ErrSkillSlotLocked
	}
	if hero.GetEquipSkillBySlot(slot) != nil {
		return nil, ErrSkillSlotOccupied
	}

	hs := role.GetSkills().GetSkillByConfID(skillConfID)
	if hs == nil {
		return nil, ErrSkillNotOwned
	}
	if hs.GetEquipHeroID() != 0 {
		return nil, ErrSkillEquippedOther
	}
	if hs.GetUsedCount() >= hs.GetUseCountLimit() {
		return nil, ErrSkillUseLimitExceed
	}

	hero.EquipSkills = append(hero.EquipSkills, &pb_skill.Skill{
		ConfigId: skillConfID,
		SlotId:   slot,
		Level:    hs.GetLevel(),
	})
	hs.EquipTo(hero.ID)
	return hero.EquipSkills, nil
}

// HeroUnequipSkill 拆卸技能：移除槽位技能 + 清空技能库装配记录 + 按等级比例返还升级资源
//
// 返还：初始等级 1 起，每升级 1 级返还 1/2 该级升级道具（向下取整）。
func HeroUnequipSkill(role *game_roles.Role, hero *role_heroes.RoleHero, slot int32) ([]*pb_skill.Skill, int64, error) {
	if slot < skillEquipSlotMin || slot > skillEquipSlotMax {
		return nil, 0, ErrSkillSlotInvalid
	}
	equipped := hero.GetEquipSkillBySlot(slot)
	if equipped == nil {
		return nil, 0, ErrSkillSlotEmpty
	}

	// 移除槽位技能
	hero.EquipSkills = removeEquipSkill(hero.EquipSkills, slot)

	// 清空技能库装配记录
	if hs := role.GetSkills().GetSkillByConfID(equipped.GetConfigId()); hs != nil {
		hs.Unequip()
	}

	// 按等级返还升级道具
	refund := refundUpgradeCost(equipped.GetConfigId(), equipped.GetLevel())
	if refund > 0 {
		if err := ItemChange(role, []common_declarations.ItemUse{
			{ItemID: skillConfByID[equipped.GetConfigId()].UpgradeCost.ItemID, Count: refund},
		}, nil, common_declarations.ReasonSkill); err != nil {
			return hero.EquipSkills, 0, err
		}
	}

	return hero.EquipSkills, refund, nil
}

// HeroSkillUpgrade 技能升级：升级英雄身上（槽位）技能等级，消耗道具
//
//   - 前置：槽位有效/已装配、未满级
//   - 消耗 UpgradeCost（不足时返回携带 ItemTypeNormalNotEnough 的 error）
//   - 等级只存在槽位技能上，不同步技能库(hero_skills)等级
func HeroSkillUpgrade(role *game_roles.Role, hero *role_heroes.RoleHero, slot int32) (*pb_skill.Skill, error) {
	if slot < skillEquipSlotMin || slot > skillEquipSlotMax {
		return nil, ErrSkillSlotInvalid
	}
	equipped := hero.GetEquipSkillBySlot(slot)
	if equipped == nil {
		return nil, ErrSkillSlotEmpty
	}
	conf, ok := skillConfByID[equipped.GetConfigId()]
	if !ok {
		return nil, ErrSkillConfNotFound
	}
	if equipped.GetLevel() >= conf.MaxLevel {
		return nil, ErrSkillMaxLevel
	}

	if err := ItemChange(role, nil, []common_declarations.ItemUse{conf.UpgradeCost}, common_declarations.ReasonSkill); err != nil {
		return nil, err
	}

	equipped.Level = equipped.GetLevel() + 1
	return equipped, nil
}

// removeEquipSkill 从技能槽移除指定槽位
func removeEquipSkill(list []*pb_skill.Skill, slot int32) []*pb_skill.Skill {
	out := list[:0]
	for _, s := range list {
		if s.GetSlotId() != slot {
			out = append(out, s)
		}
	}
	return out
}

// refundUpgradeCost 按已升级等级计算返还的升级道具数量
//
// 初始等级 1，升级次数 = level-1；每级返还 UpgradeCost.Count/2（向下取整）。
func refundUpgradeCost(skillConfID, level int32) int64 {
	conf, ok := skillConfByID[skillConfID]
	if !ok {
		return 0
	}
	upgraded := int64(level - 1)
	if upgraded <= 0 {
		return 0
	}
	return (conf.UpgradeCost.Count * upgraded) / int64(skillUnequipRefund)
}
