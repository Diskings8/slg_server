package building_handler

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerBuildingBuild 建造建筑 (1000011)
//
// 主城/分城/军事建筑统一走此入口，type 字段区分。
// 资源消耗走 BuildingCostI（预留接口，TODO: 接入货币道具后实现）。
func HandlerBuildingBuild(ctx context.Context, roleID uint64, req *pb_city.BuildingBuildReq, resp *pb_city.BuildingBuildResp) rpc_results.ResultI {
	// 参数格式校验（快速失败，不加载角色）
	if req.GetMapId() < 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid map_id")
	}

	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	buildingID, result := game_logics.BuildingBuild(role, roleID, req)
	if result != nil {
		return result
	}

	poller.Save() // 打脏标记，异步保存

	resp.Building = game_logics.BuildingGetPb(role, buildingID)
	return nil
}

// HandlerBuildingList 查询建筑列表 (1000012)
func HandlerBuildingList(ctx context.Context, roleID uint64, req *pb_city.BuildingListReq, resp *pb_city.BuildingListResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	resp.Buildings = game_logics.BuildingListPb(role)
	return nil
}
