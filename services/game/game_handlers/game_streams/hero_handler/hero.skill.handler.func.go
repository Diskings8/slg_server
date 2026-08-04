package hero_handler

import (
	"context"
	"errors"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroSkillUpgrade 技能升级 (1000004)
//
// 升级英雄身上（EquipSkills 槽位）技能等级，消耗道具。
func HandlerHeroSkillUpgrade(ctx context.Context, roleID uint64, req *pb_hero.HeroSkillUpgradeReq, resp *pb_hero.HeroSkillUpgradeResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	skill, logicErr := game_logics.HeroSkillUpgrade(role, hero, req.GetSlotId())
	if logicErr != nil {
		return skillLogicError(logicErr)
	}

	poller.Save()
	resp.Skill = skill
	return nil
}

// HandlerHeroEquipSkill 装配技能 (1000016)
//
// 将角色技能库技能放入英雄技能槽（校验槽位可用性/技能库状态）。
func HandlerHeroEquipSkill(ctx context.Context, roleID uint64, req *pb_hero.HeroEquipSkillReq, resp *pb_hero.HeroEquipSkillResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	skills, logicErr := game_logics.HeroEquipSkill(role, hero, req.GetSlotId(), req.GetSkillConfId())
	if logicErr != nil {
		return skillLogicError(logicErr)
	}

	poller.Save()
	resp.Skills = skills
	return nil
}

// HandlerHeroUnequipSkill 拆卸技能 (1000017)
//
// 从英雄技能槽移除技能，清空装配记录，按等级返还部分升级资源。
func HandlerHeroUnequipSkill(ctx context.Context, roleID uint64, req *pb_hero.HeroUnequipSkillReq, resp *pb_hero.HeroUnequipSkillResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	skills, refund, logicErr := game_logics.HeroUnequipSkill(role, hero, req.GetSlotId())
	if logicErr != nil {
		return skillLogicError(logicErr)
	}

	poller.Save()
	resp.Skills = skills
	resp.RefundCount = refund
	return nil
}

// skillLogicError 技能逻辑错误映射：
//
//	业务校验类（槽位/技能库状态）→ ParamError；道具消耗不足等 ResultI → 原样透传；兜底 Failed。
func skillLogicError(logicErr error) rpc_results.ResultI {
	switch {
	case errors.Is(logicErr, game_logics.ErrSkillConfNotFound),
		errors.Is(logicErr, game_logics.ErrSkillSlotInvalid),
		errors.Is(logicErr, game_logics.ErrSkillSlotLocked),
		errors.Is(logicErr, game_logics.ErrSkillSlotOccupied),
		errors.Is(logicErr, game_logics.ErrSkillSlotEmpty),
		errors.Is(logicErr, game_logics.ErrSkillNotOwned),
		errors.Is(logicErr, game_logics.ErrSkillEquippedOther),
		errors.Is(logicErr, game_logics.ErrSkillUseLimitExceed),
		errors.Is(logicErr, game_logics.ErrSkillMaxLevel):
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, logicErr.Error())
	default:
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 道具不足等业务错误原样透传（含 ItemTypeNormalNotEnough）
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("skill logic failed: %s", logicErr.Error()))
	}
}
