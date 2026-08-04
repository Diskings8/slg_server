package hero_handler

import (
	"context"
	"errors"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	pb_confs "server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroUpgradeLevel 英雄获得经验并升级 (1000002)
//
// 给英雄累加经验，满足升级所需经验则升级（每10级发自由属性点）。
func HandlerHeroUpgradeLevel(ctx context.Context, roleID uint64, req *pb_hero.HeroUpgradeLevelReq, resp *pb_hero.HeroUpgradeLevelResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	newLevel, logicErr := game_logics.HeroAddExp(hero, req.GetExp())
	if logicErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, logicErr.Error())
	}

	poller.Save()

	resp.HeroId = req.GetHeroId()
	resp.Level = newLevel
	resp.AttrPoint = hero.GetAttrPoint()
	return nil
}

// HandlerHeroCultivate 英雄加点 (1000003)
//
// 消耗 1 点自由属性点，给指定属性（除拆迁值外）加 1 点。
func HandlerHeroCultivate(ctx context.Context, roleID uint64, req *pb_hero.HeroCultivateReq, resp *pb_hero.HeroCultivateResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
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
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId()))
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	if logicErr := game_logics.HeroUpgradeStar(role, hero); logicErr != nil {
		switch {
		case errors.Is(logicErr, game_logics.ErrHeroStarFull),
			errors.Is(logicErr, game_logics.ErrHeroNoConsumeCard):
			return rpc_results.Error(pb_error_code.ErrorCode_ParamError, logicErr.Error())
		default:
			return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("upgrade star failed: %s", logicErr.Error()))
		}
	}

	poller.Save()
	resp.HeroId = req.GetHeroId()
	resp.StarStage = hero.GetStarStage()
	return nil
}
