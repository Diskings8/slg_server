package game_logics

import (
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_models"
)

// BuildingBuild 建造建筑
//
// 主城/分城/军事建筑统一走此入口，type 字段区分。
// 资源消耗走 BuildingCostI（预留接口，TODO: 接入货币道具后实现）。
// 返回新建建筑 ID（用于后续落位/查询）。
func BuildingBuild(role *game_roles.Role, roleID uint64, req *pb_city.BuildingBuildReq) (uint64, rpc_results.ResultI) {
	// 仅允许建造角色建筑类型
	if !isRoleBuildingType(req.GetType()) {
		return 0, rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid building type")
	}

	// TODO: 资源消耗检查与扣除（预留 BuildingCostI，接入货币道具后实现）
	// cost := buildingCostOf(req.GetType())
	// if err := cost.CheckCost(role); err != nil { ... }
	// if err := cost.DeductCost(role); err != nil { ... }

	// 创建建筑记录
	building := &game_models.RoleBuilding{
		RoleID:    roleID,
		Type:      req.GetType(),
		Footprint: footprintOf(req.GetType()),
		MapID:     req.GetMapId(),
		Level:     1,
		State:     pb_city.BuildingState_Completed, // TODO: 引入建造时长后置为 Constructing
		CityID:    req.GetCityId(),                 // 兵营等附属建筑归属城市；主城/分城为 0
	}
	building.ID = snowflakes.GenUUID()

	role.GetBuildings().AddBuilding(building)

	// 城市建成 → 按校场等级分配队列（每级 1 个队列）
	if isCity(req.GetType()) {
		queueNum := drillLevelOf(req.GetType())
		for range queueNum {
			role.GetFormations().CreateFormation(roleID, building.ID)
		}
	}

	// TODO: 地图落位（Phase 4：调用 cores 放置建筑到地图格）

	return building.ID, nil
}

// BuildingGetPb 获取单个建筑（proto）
func BuildingGetPb(role *game_roles.Role, buildingID uint64) *pb_city.RoleBuildingInfo {
	building := role.GetBuildings().GetBuilding(buildingID)
	if building == nil {
		return nil
	}
	return formatBuilding(building.RoleBuilding)
}

// BuildingListPb 建筑列表（proto）
func BuildingListPb(role *game_roles.Role) []*pb_city.RoleBuildingInfo {
	list := make([]*pb_city.RoleBuildingInfo, 0)
	for _, modelOne := range role.GetBuildings().List {
		list = append(list, formatBuilding(modelOne))
	}
	return list
}

//-------------------------------

// isCity 是否城市类建筑（主城/分城，拥有校场）
func isCity(t pb_city.BuildingType) bool {
	return t == pb_city.BuildingType_RoleMainCity || t == pb_city.BuildingType_RoleBranchCity
}

// drillLevelOf 城市校场等级（TODO: 接入配置，不同玩家城市不同）
func drillLevelOf(t pb_city.BuildingType) uint32 {
	return 1 // 默认 1 级 → 1 个队列
}

// isRoleBuildingType 是否为角色可建造的建筑类型
func isRoleBuildingType(t pb_city.BuildingType) bool {
	switch t {
	case pb_city.BuildingType_RoleMainCity,
		pb_city.BuildingType_RoleBranchCity,
		pb_city.BuildingType_RoleMilitary,
		pb_city.BuildingType_RoleBarracks:
		return true
	}
	return false
}

// footprintOf 按类型返回默认占地（TODO: 接入配置）
func footprintOf(t pb_city.BuildingType) pb_city.BuildingFootprint {
	switch t {
	case pb_city.BuildingType_RoleMainCity:
		return pb_city.BuildingFootprint_Footprint9
	case pb_city.BuildingType_RoleBranchCity:
		return pb_city.BuildingFootprint_Footprint9 // TODO: 按分城功能配置 4/9
	case pb_city.BuildingType_RoleMilitary:
		return pb_city.BuildingFootprint_Footprint4 // TODO: 确认军事建筑占地
	case pb_city.BuildingType_RoleBarracks:
		return pb_city.BuildingFootprint_Footprint4 // 兵营 2×2（占位可调）
	default:
		return pb_city.BuildingFootprint_Footprint_None
	}
}

// formatBuilding 模型 → proto
func formatBuilding(b *game_models.RoleBuilding) *pb_city.RoleBuildingInfo {
	if b == nil {
		return nil
	}
	return &pb_city.RoleBuildingInfo{
		Id:        b.ID,
		Type:      b.Type,
		Footprint: b.Footprint,
		MapId:     b.MapID,
		Level:     b.Level,
		State:     b.State,
		CityId:    b.CityID,
	}
}
