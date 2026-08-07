package building_handler

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerBuildingBuild 建造建筑 (1000011)
//
// 主城/军事/兵营/校场等统一入口，type 字段区分。
// 消耗资源 + 建造时长（Constructing + EndTimeUx 惰性结算）。
func HandlerBuildingBuild(ctx context.Context, roleID uint64, req *pb_city.BuildingBuildReq, resp *pb_city.BuildingBuildResp) rpc_results.ResultI {
	// 参数格式校验（快速失败，不加载角色）
	if req.GetMapId() < 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid map_id")
	}

	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if game_logics.SettleBuildings(role, roleID) {
		poller.Save() // 先结算存量建造（幂等）
	}

	buildingID, result := game_logics.BuildingBuild(role, roleID, req)
	if result != nil {
		return result
	}

	poller.Save() // 打脏标记，异步保存

	resp.Building = game_logics.BuildingGetPb(role, buildingID)
	return nil
}

// HandlerBuildingList 查询建筑列表 (1000012)
//
// 写路径（需到期结算落库 + 同步队列），按 city_id 过滤。
func HandlerBuildingList(ctx context.Context, roleID uint64, req *pb_city.BuildingListReq, resp *pb_city.BuildingListResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if game_logics.SettleBuildings(role, roleID) {
		poller.Save()
	}

	resp.Buildings = game_logics.BuildingListPb(role, req.GetCityId())
	return nil
}

// HandlerBuildingUpgrade 升级建筑 (1000036)
func HandlerBuildingUpgrade(ctx context.Context, roleID uint64, req *pb_city.BuildingUpgradeReq, resp *pb_city.BuildingUpgradeResp) rpc_results.ResultI {
	if req.GetBuildingId() <= 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid building_id")
	}

	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if game_logics.SettleBuildings(role, roleID) {
		poller.Save()
	}

	if result := game_logics.BuildingUpgrade(role, roleID, req); result != nil {
		return result
	}

	poller.Save()

	resp.Building = game_logics.BuildingGetPb(role, req.GetBuildingId())
	return nil
}

// HandlerBuildingGet 查询单个建筑 (1000037)
func HandlerBuildingGet(ctx context.Context, roleID uint64, req *pb_city.BuildingGetReq, resp *pb_city.BuildingGetResp) rpc_results.ResultI {
	if req.GetBuildingId() <= 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid building_id")
	}

	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if game_logics.SettleBuildings(role, roleID) {
		poller.Save()
	}

	resp.Building = game_logics.BuildingGetPb(role, req.GetBuildingId())
	if resp.Building == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_BuildingNotFound, "building not found")
	}
	return nil
}
