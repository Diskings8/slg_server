package formation_handler

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerFormationField 上阵英雄到队列 (1000008)
func HandlerFormationField(ctx context.Context, roleID uint64, req *pb_maps_march.FormationFieldReq, resp *pb_maps_march.FormationFieldResp) rpc_results.ResultI {
	// 参数格式校验（快速失败，不加载角色）
	if req.GetCityId() == 0 || req.GetFormationId() == 0 || req.GetSlotPos() < 1 || req.GetHeroId() == 0 || req.GetSoldierNum() == 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid params")
	}

	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if result := game_logics.FormationFieldHero(role, req); result != nil {
		return result
	}
	poller.Save() // 打脏标记，异步保存
	resp.Formation = game_logics.FormationGetPb(role, req.GetFormationId())
	return nil
}

// HandlerFormationRemove 下阵英雄 (1000009)
func HandlerFormationRemove(ctx context.Context, roleID uint64, req *pb_maps_march.FormationRemoveReq, resp *pb_maps_march.FormationRemoveResp) rpc_results.ResultI {
	// 参数格式校验
	if req.GetFormationId() == 0 || req.GetSlotPos() < 1 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid params")
	}

	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if result := game_logics.FormationRemoveHero(role, req); result != nil {
		return result
	}
	poller.Save() // 打脏标记，异步保存
	resp.Formation = game_logics.FormationGetPb(role, req.GetFormationId())
	return nil
}

// HandlerFormationList 查询编队 (1000010)；city_id 可选，0 = 全部建筑
func HandlerFormationList(ctx context.Context, roleID uint64, req *pb_maps_march.FormationListReq, resp *pb_maps_march.FormationListResp) rpc_results.ResultI {
	// 只读：免锁快照，无需 Release
	role, result := game_roles.GetCopy(roleID)
	if result != nil {
		return result
	}

	resp.Formations = game_logics.FormationListPb(role, req.GetCityId())
	return nil
}
