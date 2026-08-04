package march_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_logics"
)

// HandlerMarchCreate 创建行军 (1000006)
//
// 从已上阵队伍（formation_id）取英雄+士兵构造出征队伍，调用 worldmap 创建行军。
func HandlerMarchCreate(ctx context.Context, roleID uint64, req *pb_maps_march.MarchCreateReq, resp *pb_maps_march.MarchCreateResp) rpc_results.ResultI {
	if req.GetToMapId() < 0 || req.GetMarchType() < 1 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid march params")
	}

	// 只读（MarchBuildTeam 只取编队+英雄快照，不修改角色）：免锁快照，无需 Release
	role, result := game_role_handler.GetCopy(roleID)
	if result != nil {
		return result
	}

	// 构造出征队伍（英雄快照 + 士兵）
	teamSlots, marchResult := game_logics.MarchBuildTeam(role, req)
	if marchResult != nil {
		return marchResult
	}

	// 调用 worldmap 创建行军
	createRsp, callErr := game_rpc_clients.WorldMap().CreateMarch(ctx, &pb_worldmap.CreateMarchReq{
		RoleId:    roleID,
		FromMapId: req.GetFromMapId(),
		ToMapId:   req.GetToMapId(),
		MarchType: req.GetMarchType(),
		TeamSlots: teamSlots,
		BaseSpeed: 100,
	})
	if callErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("create march failed: %s", callErr.Error()))
	}

	resp.MarchId = createRsp.GetMarchId()
	resp.EndTime = createRsp.GetEndTime()
	return nil
}
