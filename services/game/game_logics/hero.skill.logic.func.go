package game_logics

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// heroSkillSlotUnlocked 英雄技能槽位可用性判断（只做条件判断）
//
//   - index0：默认技能槽，始终可用
//   - index1：英雄等级 ≥ 阈值
//   - index2：英雄等级 ≥ 阈值 且 已觉醒
func heroSkillSlotUnlocked(hero *role_heroes.RoleHero, slot int32) bool {
	sc := game_conf.Load().Skill
	switch slot {
	case 1:
		return hero.GetLevel() >= sc.Slot1UnlockLv
	case 2:
		return hero.GetLevel() >= sc.Slot2UnlockLv && hero.GetIsAwakened()
	}
	return slot == sc.SlotDefault
}

// HeroEquipSkill 装配技能：将角色技能库技能放入英雄技能槽
//
// 前置：槽位有效/已解锁/为空、技能库存在该技能、未装配他处、装配次数未用尽。
// 装配后技能库记录被装配英雄 + 装配次数 +1；槽位技能初始等级 = 技能库等级（升级不同步）。
func HeroEquipSkill(role *game_roles.Role, hero *role_heroes.RoleHero, slot, skillConfID int32) ([]*pb_skill.Skill, error) {
	if slot < game_conf.Load().Skill.SlotEquipMin || slot > game_conf.Load().Skill.SlotEquipMax {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotInvalid, "skill slot invalid")
	}
	if !heroSkillSlotUnlocked(hero, slot) {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotLocked, "skill slot locked")
	}
	if hero.GetEquipSkillBySlot(slot) != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotOccupied, "skill slot occupied")
	}

	hs := role.GetSkills().GetSkillByConfID(skillConfID)
	if hs == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillNotOwned, "skill not owned")
	}
	if hs.GetEquipHeroID() != 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillEquippedOther, "skill equipped by other hero")
	}
	if hs.GetUsedCount() >= hs.GetUseCountLimit() {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillUseLimitExceed, "skill use count limit exceeded")
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
	if slot < game_conf.Load().Skill.SlotEquipMin || slot > game_conf.Load().Skill.SlotEquipMax {
		return nil, 0, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotInvalid, "skill slot invalid")
	}
	equipped := hero.GetEquipSkillBySlot(slot)
	if equipped == nil {
		return nil, 0, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotEmpty, "skill slot empty")
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
		if conf, ok := game_conf.Load().Skill.GetSkillConf(equipped.GetConfigId()); ok {
			if err := ItemChange(role, []common_declarations.ItemUse{
				{ItemID: conf.UpgradeCost.ItemID, Count: refund},
			}, nil, common_declarations.ReasonSkill); err != nil {
				return hero.EquipSkills, 0, err
			}
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
	if slot < game_conf.Load().Skill.SlotEquipMin || slot > game_conf.Load().Skill.SlotEquipMax {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotInvalid, "skill slot invalid")
	}
	equipped := hero.GetEquipSkillBySlot(slot)
	if equipped == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillSlotEmpty, "skill slot empty")
	}
	conf, ok := game_conf.Load().Skill.GetSkillConf(equipped.GetConfigId())
	if !ok {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillConfNotFound, "skill conf not found")
	}
	if equipped.GetLevel() >= conf.MaxLevel {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_HeroSkillMaxLevel, "skill already max level")
	}

	if err := ItemChange(role, nil, []common_declarations.ItemUse{conf.UpgradeCost}, common_declarations.ReasonSkill); err != nil {
		return nil, err
	}

	// 记录养成消耗（本次升级消耗的道具 config + 数量）
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateSkill, []*pb_common.Int32KV{
		{Key: int32(conf.UpgradeCost.ItemID), Val: int32(conf.UpgradeCost.Count)},
	})

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
	conf, ok := game_conf.Load().Skill.GetSkillConf(skillConfID)
	if !ok {
		return 0
	}
	upgraded := int64(level - 1)
	if upgraded <= 0 {
		return 0
	}
	return (conf.UpgradeCost.Count * upgraded) / int64(game_conf.Load().Skill.UnequipRefund)
}
