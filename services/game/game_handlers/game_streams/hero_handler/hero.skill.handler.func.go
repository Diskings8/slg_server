package hero_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroSkillCollectionActivate 技能收藏激活 (1000019)
//
// 消耗客户端选定的一张英雄卡推进收集进度，全部达标解锁对应技能并发放到技能库。
func HandlerHeroSkillCollectionActivate(ctx context.Context, roleID uint64, req *pb_skill.SkillCollectionActivateReq, resp *pb_skill.SkillCollectionActivateResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if req.GetSkillConfId() <= 0 || req.GetHeroId() <= 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid params")
	}

	if logicErr := game_logics.SkillCollectionActivate(role, req.GetSkillConfId(), req.GetHeroId()); logicErr != nil {
		return skillLogicError(logicErr)
	}

	poller.Save()
	if collection := role.GetSkillCollections().GetBySkillConfID(req.GetSkillConfId()); collection != nil {
		resp.Collection = collection.Format2Pb()
	}
	return nil
}

// HandlerHeroSkillUpgrade 技能升级 (1000004)
//
// 升级英雄身上（EquipSkills 槽位）技能等级，消耗道具。
func HandlerHeroSkillUpgrade(ctx context.Context, roleID uint64, req *pb_hero.HeroSkillUpgradeReq, resp *pb_hero.HeroSkillUpgradeResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
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
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
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
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
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

// skillLogicError 技能逻辑错误处理
//
// 逻辑层已统一返回 rpc_results.ResultI（带专属错误码），此处直接透传；非 ResultI（意外错误）用 Failed 兜底。
func skillLogicError(logicErr error) rpc_results.ResultI {
	if r, ok := logicErr.(rpc_results.ResultI); ok {
		return r
	}
	return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("skill logic failed: %s", logicErr.Error()))
}
