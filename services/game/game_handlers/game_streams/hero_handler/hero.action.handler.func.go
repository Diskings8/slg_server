package hero_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroCultivate 英雄加点 (1000003)
//
// 消耗 1 点自由属性点，给指定属性（除拆迁值外）加 1 点。
func HandlerHeroCultivate(ctx context.Context, roleID uint64, req *pb_hero.HeroCultivateReq, resp *pb_hero.HeroCultivateResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	attr, logicErr := game_logics.HeroCultivate(hero, req.GetCultivateType())
	if logicErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("cultivate failed: %s", logicErr.Error()))
	}

	poller.Save()

	resp.HeroId = req.GetHeroId()
	resp.CultivateType = req.GetCultivateType()
	resp.Attr = attr
	resp.AttrPoint = hero.GetAttrPoint()
	return nil
}

// HandlerHeroUpgradeStar 英雄升星 (1000018)
//
// 消耗一张同配置英雄卡升 1 星，星级上限常量配置，被消耗卡配置记录进养成消耗记录。
func HandlerHeroUpgradeStar(ctx context.Context, roleID uint64, req *pb_hero.HeroUpgradeStarReq, resp *pb_hero.HeroUpgradeStarResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	if logicErr := game_logics.HeroUpgradeStar(role, hero); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 满星/无消耗卡等专属错误码透传
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("upgrade star failed: %s", logicErr.Error()))
	}

	poller.Save()
	resp.HeroId = req.GetHeroId()
	resp.StarStage = hero.GetStarStage()
	return nil
}

// HandlerHeroAwaken 英雄觉醒 (1000043)
//
// 消耗觉醒资源 → 置 IsAwakened，解锁第三技能槽（等级不足/已觉醒返回专属错误码）。
func HandlerHeroAwaken(ctx context.Context, roleID uint64, req *pb_hero.HeroAwakenReq, resp *pb_hero.HeroAwakenResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	if logicErr := game_logics.HeroAwaken(role, hero); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 已觉醒/等级不足等专属错误码透传
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("awaken failed: %s", logicErr.Error()))
	}

	poller.Save()
	resp.HeroId = req.GetHeroId()
	resp.Awakened = hero.GetIsAwakened()
	return nil
}

// HandlerHeroLock 锁定英雄 (1000020)
//
// 锁定后英雄不可被消耗（如作为升星消耗卡）。
func HandlerHeroLock(ctx context.Context, roleID uint64, req *pb_hero.HeroLockReq, resp *pb_hero.HeroLockResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	game_logics.HeroLock(hero)
	poller.Save()
	resp.HeroId = req.GetHeroId()
	resp.IsLocked = true
	return nil
}

// HandlerHeroUnlock 解锁英雄 (1000021)
func HandlerHeroUnlock(ctx context.Context, roleID uint64, req *pb_hero.HeroUnlockReq, resp *pb_hero.HeroUnlockResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	game_logics.HeroUnlock(hero)
	poller.Save()
	resp.HeroId = req.GetHeroId()
	resp.IsLocked = false
	return nil
}
