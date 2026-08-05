package recruit_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_recruit"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerRecruitPoolsInfo 查询所有抽卡池信息 (1000020)
//
// 只读：免锁快照，无需 Release。
func HandlerRecruitPoolsInfo(ctx context.Context, roleID uint64, req *pb_recruit.RecruitPoolsInfoReq, resp *pb_recruit.RecruitPoolsInfoResp) rpc_results.ResultI {
	role, result := game_role_handler.GetCopy(roleID)
	if result != nil {
		return result
	}

	resp.Pools = game_logics.RecruitPoolsInfo(role)
	return nil
}

// HandlerRecruit 单抽/十连 (1000021)
//
// 消耗顺序：每日免费 → 抽卡券 → 金币。产出英雄卡/道具，返回最新池状态。
func HandlerRecruit(ctx context.Context, roleID uint64, req *pb_recruit.RecruitReq, resp *pb_recruit.RecruitResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	result, logicErr := game_logics.Recruit(role, req.GetId(), req.GetTimes())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 池不存在/次数非法/券金币不足等专属错误码透传
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("recruit failed: %s", logicErr.Error()))
	}

	poller.Save()

	resp.CostType = result.CostType
	resp.Rewards = result.Rewards
	resp.Pools = game_logics.RecruitPoolsInfo(role)
	return nil
}

// HandlerRecruitSetWish 设置心愿英雄 (1000022)
func HandlerRecruitSetWish(ctx context.Context, roleID uint64, req *pb_recruit.RecruitSetWishReq, resp *pb_recruit.RecruitSetWishResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if logicErr := game_logics.RecruitSetWish(role, req.GetId(), req.GetChooseHero()); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("set wish failed: %s", logicErr.Error()))
	}

	poller.Save()

	for _, pool := range game_logics.RecruitPoolsInfo(role) {
		if pool.GetId() == req.GetId() {
			resp.Pool = pool
			break
		}
	}
	return nil
}

// HandlerRecruitDrawWish 领取心愿英雄卡 (1000023)
func HandlerRecruitDrawWish(ctx context.Context, roleID uint64, req *pb_recruit.RecruitDrawWishReq, resp *pb_recruit.RecruitDrawWishResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	hero, logicErr := game_logics.RecruitDrawWish(role, req.GetId())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("draw wish failed: %s", logicErr.Error()))
	}

	poller.Save()

	resp.HeroId = int32(hero.GetID())
	for _, pool := range game_logics.RecruitPoolsInfo(role) {
		if pool.GetId() == req.GetId() {
			resp.Pool = pool
			break
		}
	}
	return nil
}
