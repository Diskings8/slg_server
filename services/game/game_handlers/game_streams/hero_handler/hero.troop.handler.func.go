package hero_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerHeroTroopTransform 兵种转化 (1000013)
//
// x 等级后选择已解锁的派生兵种类型转化（消耗资源）。
func HandlerHeroTroopTransform(ctx context.Context, roleID uint64, req *pb_hero.HeroTroopTransformReq, resp *pb_hero.HeroTroopTransformResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	logicErr := game_logics.HeroTroopTransform(hero, req.GetTroopTypeId())
	if logicErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("transform failed: %s", logicErr.Error()))
	}

	poller.Save()

	resp.HeroId = req.GetHeroId()
	resp.CurTroopTypeId = hero.GetCurTroopTypeID()
	return nil
}

// HandlerHeroTroopUnlock 兵种扩展 (1000014)
//
// 使用道具解锁当前英雄的新可选派生兵种类型。
func HandlerHeroTroopUnlock(ctx context.Context, roleID uint64, req *pb_hero.HeroTroopUnlockReq, resp *pb_hero.HeroTroopUnlockResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero := role.GetHeroes().GetHero(req.GetHeroId())
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	logicErr := game_logics.HeroTroopUnlock(role, hero, req.GetTroopTypeId())
	if logicErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, fmt.Sprintf("unlock failed: %s", logicErr.Error()))
	}

	poller.Save()

	resp.HeroId = req.GetHeroId()
	resp.TroopTypeId = req.GetTroopTypeId()
	resp.Troops = hero.Troops
	return nil
}
