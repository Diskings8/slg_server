package handler_servers

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroList 获取英雄列表 (1000001)
func HandlerHeroList(ctx context.Context, roleID uint64, req *pb_hero.HeroListReq, resp *pb_hero.HeroListResp) rpc_results.ResultI {
	_ = req

	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	resp.Heroes = role.GetHeroes().Format2Pb()
	return nil
}

// HandlerHeroUpgradeLevel 英雄升级 (1000002)
func HandlerHeroUpgradeLevel(ctx context.Context, roleID uint64, req *pb_hero.HeroUpgradeLevelReq, resp *pb_hero.HeroUpgradeLevelResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	hero, ok := role.GetHeroes().Mem.Load(pb_confs.ItemID(req.GetHeroId()))
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("hero not found, id: %d", req.GetHeroId()))
	}

	newLevel, err := game_logics.HeroLevelUp(hero)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, err.Error())
	}

	resp.HeroId = req.GetHeroId()
	resp.Level = newLevel

	loggers.Logger.Info(fmt.Sprintf("HeroUpgradeLevel ok: roleID=%d, heroID=%d, level=%d", roleID, req.GetHeroId(), newLevel))
	return nil
}

// HandlerHeroCultivate 英雄培养 (1000003)
func HandlerHeroCultivate(ctx context.Context, roleID uint64, req *pb_hero.HeroCultivateReq, resp *pb_hero.HeroCultivateResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	hero, ok := role.GetHeroes().Mem.Load(pb_confs.ItemID(req.GetHeroId()))
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("hero not found, id: %d", req.GetHeroId()))
	}

	attr, err := game_logics.HeroCultivate(hero, req.GetCultivateType())
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, err.Error())
	}

	resp.HeroId = req.GetHeroId()
	resp.CultivateType = req.GetCultivateType()
	resp.Attr = attr

	loggers.Logger.Info(fmt.Sprintf("HeroCultivate ok: roleID=%d, heroID=%d, type=%d", roleID, req.GetHeroId(), req.GetCultivateType()))
	return nil
}
