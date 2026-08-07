package game_logics

import (
	"time"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_models"
)

// reasonBuilding 建造/升级建筑消耗原因
const reasonBuilding = "building"

// itemChangeResult 执行道具消耗并转换为 ResultI（ItemChange 返回 error，实际是 ResultI）
func itemChangeResult(role *game_roles.Role, addItems, useItems []common_declarations.ItemUse) rpc_results.ResultI {
	if err := ItemChange(role, addItems, useItems, reasonBuilding); err != nil {
		if res, ok := err.(rpc_results.ResultI); ok {
			return res
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, err.Error())
	}
	return nil
}

// BuildingBuild 建造建筑
//
// 统一走此入口，type 字段区分。消耗资源（ItemChange）+ 建造时长（Constructing + EndTimeUx 惰性结算）。
// 返回新建建筑 ID。
func BuildingBuild(role *game_roles.Role, roleID uint64, req *pb_city.BuildingBuildReq) (uint64, rpc_results.ResultI) {
	return buildingBuild(role, roleID, req, false)
}

// BuildMainCityInstant 建角即时建主城（跳过建造时长与消耗，直接 Completed）
func BuildMainCityInstant(role *game_roles.Role, roleID uint64, req *pb_city.BuildingBuildReq) (uint64, rpc_results.ResultI) {
	return buildingBuild(role, roleID, req, true)
}

func buildingBuild(role *game_roles.Role, roleID uint64, req *pb_city.BuildingBuildReq, instant bool) (uint64, rpc_results.ResultI) {
	t := req.GetType()
	if !isRoleBuildingType(t) {
		return 0, rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid building type")
	}

	conf, ok := game_conf.Load().Building.GetBuilding(t)
	if !ok {
		return 0, rpc_results.Error(pb_error_code.ErrorCode_BuildingConfNotFound, "building conf not found")
	}

	// 唯一性：城市类不限数量；附属类同 city 同类型唯一
	if role.GetBuildings().BuildingExists(t, req.GetCityId()) {
		return 0, rpc_results.Error(pb_error_code.ErrorCode_BuildingAlreadyExists, "building already exists")
	}

	// 资源消耗（统一 ItemChange，含余额检查；instant 建角不扣）
	if !instant && len(conf.BuildCost) > 0 {
		if err := itemChangeResult(role, nil, conf.BuildCost); err != nil {
			return 0, err
		}
	}

	building := &game_models.RoleBuilding{
		RoleID:    roleID,
		Type:      t,
		Footprint: conf.Footprint,
		MapID:     req.GetMapId(),
		Level:     0,
		State:     pb_city.BuildingState_Constructing,
		CityID:    req.GetCityId(),
		NextLevel: 1,
	}
	building.ID = snowflakes.GenUUID()

	if instant || conf.BuildTimeUx <= 0 {
		completeBuilding(building) // Level=NextLevel; NextLevel=0; State=Completed
	} else {
		building.EndTimeUx = time.Now().Unix() + conf.BuildTimeUx
	}

	role.GetBuildings().AddBuilding(building)

	// 城市已建成 → 落校场 + 同步队列
	if isCity(t) && building.State == pb_city.BuildingState_Completed {
		finalizeCity(role, roleID, building)
	}

	return building.ID, nil
}

// BuildingUpgrade 升级建筑
func BuildingUpgrade(role *game_roles.Role, roleID uint64, req *pb_city.BuildingUpgradeReq) rpc_results.ResultI {
	building := role.GetBuildings().GetBuilding(req.GetBuildingId())
	if building == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_BuildingNotFound, "building not found")
	}

	settleBuilding(role, roleID, building.RoleBuilding) // 到期先结算（幂等）
	if building.State == pb_city.BuildingState_Constructing {
		return rpc_results.Error(pb_error_code.ErrorCode_BuildingConstructing, "building is constructing")
	}

	conf, ok := game_conf.Load().Building.GetBuilding(building.Type)
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_BuildingConfNotFound, "building conf not found")
	}
	if building.Level >= conf.MaxLevel {
		return rpc_results.Error(pb_error_code.ErrorCode_BuildingMaxLevel, "building reach max level")
	}

	cost, _ := game_conf.Load().Building.UpgradeCost(building.Type, building.Level)
	if len(cost) > 0 {
		if err := itemChangeResult(role, nil, cost); err != nil {
			return err
		}
	}

	building.NextLevel = building.Level + 1
	building.State = pb_city.BuildingState_Constructing
	building.EndTimeUx = time.Now().Unix() + game_conf.Load().Building.UpgradeTime(building.Type, building.Level)
	return nil
}

// SettleBuildings 结算所有到期的建筑；返回是否有变化（调用方决定是否 poller.Save）
func SettleBuildings(role *game_roles.Role, roleID uint64) bool {
	changed := false
	for _, b := range role.GetBuildings().List {
		if settleBuilding(role, roleID, b) {
			changed = true
		}
	}
	return changed
}

// settleBuilding 单建筑到期结算：now >= EndTimeUx → Level=NextLevel, 置 Completed
// 完成回调：城市→finalizeCity（落校场+同步队列）；校场→syncDrillQueue
func settleBuilding(role *game_roles.Role, roleID uint64, b *game_models.RoleBuilding) bool {
	if b.State != pb_city.BuildingState_Constructing {
		return false
	}
	now := time.Now().Unix()
	if b.NextLevel == 0 { // 数据异常卡在建中 → 兜底完成
		b.Level, b.NextLevel, b.State, b.EndTimeUx = 1, 0, pb_city.BuildingState_Completed, 0
		return true
	}
	if now < b.EndTimeUx {
		return false
	}
	b.Level = b.NextLevel
	b.NextLevel = 0
	b.State = pb_city.BuildingState_Completed
	b.EndTimeUx = 0
	if isCity(b.Type) {
		finalizeCity(role, roleID, b)
	} else if b.Type == pb_city.BuildingType_RoleDrill {
		syncDrillQueue(role, roleID, b.CityID)
	}
	return true
}

// finalizeCity 城市建成：自动建 1 级校场（Completed）+ 同步队列
func finalizeCity(role *game_roles.Role, roleID uint64, city *game_models.RoleBuilding) {
	if role.GetBuildings().GetDrillByCity(city.ID) != nil {
		syncDrillQueue(role, roleID, city.ID)
		return
	}
	conf, ok := game_conf.Load().Building.GetBuilding(pb_city.BuildingType_RoleDrill)
	if !ok {
		return
	}
	drill := &game_models.RoleBuilding{
		RoleID:    roleID,
		Type:      pb_city.BuildingType_RoleDrill,
		Footprint: conf.Footprint,
		MapID:     city.MapID,
		Level:     1,
		State:     pb_city.BuildingState_Completed,
		CityID:    city.ID,
	}
	drill.ID = snowflakes.GenUUID()
	role.GetBuildings().AddBuilding(drill)
	syncDrillQueue(role, roleID, city.ID)
}

// syncDrillQueue 按校场等级配置同步队列数（本轮只增不减；缩减 TODO）
func syncDrillQueue(role *game_roles.Role, roleID, cityID uint64) {
	drill := role.GetBuildings().GetDrillByCity(cityID)
	if drill == nil {
		return
	}
	target := game_conf.Load().Building.QueueNumAtLevel(drill.Level)
	current := uint32(len(role.GetFormations().ListByCity(cityID)))
	for current < target {
		role.GetFormations().CreateFormation(roleID, cityID)
		current++
	}
}

// completeBuilding 建造完成：Level=NextLevel, NextLevel=0, State=Completed, EndTimeUx=0
func completeBuilding(building *game_models.RoleBuilding) {
	building.Level = building.NextLevel
	building.NextLevel = 0
	building.State = pb_city.BuildingState_Completed
	building.EndTimeUx = 0
}

// BuildingGetPb 获取单个建筑（proto）
func BuildingGetPb(role *game_roles.Role, buildingID uint64) *pb_city.RoleBuildingInfo {
	building := role.GetBuildings().GetBuilding(buildingID)
	if building == nil {
		return nil
	}
	return formatBuilding(building.RoleBuilding)
}

// BuildingListPb 建筑列表（proto）；cityID=0 返回全部，否则按归属城市过滤
func BuildingListPb(role *game_roles.Role, cityID uint64) []*pb_city.RoleBuildingInfo {
	list := make([]*pb_city.RoleBuildingInfo, 0)
	for _, modelOne := range role.GetBuildings().List {
		if cityID != 0 && modelOne.CityID != cityID {
			continue
		}
		list = append(list, formatBuilding(modelOne))
	}
	return list
}

//-------------------------------

// isCity 是否城市类建筑（主城，拥有校场）
func isCity(t pb_city.BuildingType) bool {
	return t == pb_city.BuildingType_RoleMainCity || t == pb_city.BuildingType_RoleBranchCity
}

// isRoleBuildingType 是否为角色可建造的建筑类型
func isRoleBuildingType(t pb_city.BuildingType) bool {
	switch t {
	case pb_city.BuildingType_RoleMainCity,
		pb_city.BuildingType_RoleBranchCity,
		pb_city.BuildingType_RoleMilitary,
		pb_city.BuildingType_RoleBarracks,
		pb_city.BuildingType_RoleDrill,
		pb_city.BuildingType_RoleWall,
		pb_city.BuildingType_RoleWarehouse,
		pb_city.BuildingType_RoleFarm,
		pb_city.BuildingType_RoleLumber,
		pb_city.BuildingType_RoleStone,
		pb_city.BuildingType_RoleIron:
		return true
	}
	return false
}

// footprintOf 按类型返回默认占地（读配置；无配置用现有默认）
func footprintOf(t pb_city.BuildingType) pb_city.BuildingFootprint {
	if conf, ok := game_conf.Load().Building.GetBuilding(t); ok {
		return conf.Footprint
	}
	return pb_city.BuildingFootprint_Footprint_None
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
		NextLevel: b.NextLevel,
		EndTimeUx: b.EndTimeUx,
	}
}
