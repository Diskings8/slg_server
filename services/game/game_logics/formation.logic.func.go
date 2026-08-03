package game_logics

import (
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	pb_confs "server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_models"
)

// maxFormationSlots 编队英雄槽位上限（TODO: 接入配置，默认 3）
const maxFormationSlots = 3

// FormationFieldHero 上阵英雄到队列
//
// 前置：队列已分配（城市校场等级）、城市存在且已完成。
func FormationFieldHero(role *game_roles.Role, req *pb_maps_march.FormationFieldReq) rpc_results.ResultI {
	// 1. 校验上阵城市存在且已完成
	city := role.GetBuildings().GetBuilding(req.GetCityId())
	if city == nil || city.State != pb_city.BuildingState_Completed {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "city not available")
	}
	// TODO: 校验校场等级符合（配置驱动）

	// 2. 校验队列已分配且归属该城市
	formation := role.GetFormations().GetFormationByID(req.GetFormationId())
	if formation == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation not found")
	}
	if formation.CityID != req.GetCityId() {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation city mismatch")
	}

	// 3. 校验英雄存在
	if hero := role.GetHeroes().GetHero(pb_confs.ItemID(req.GetHeroId())); hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero not found")
	}

	// 4. 设置槽位
	slots := setHeroSlot(formation.HeroSlots, int(req.GetSlotPos()), req.GetHeroId(), req.GetSoldierNum())
	if slots == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation slot full")
	}
	formation.HeroSlots = slots
	return nil
}

// FormationRemoveHero 下阵英雄
func FormationRemoveHero(role *game_roles.Role, req *pb_maps_march.FormationRemoveReq) rpc_results.ResultI {
	formation := role.GetFormations().GetFormationByID(req.GetFormationId())
	if formation == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation not found")
	}

	formation.HeroSlots = removeHeroSlotAt(formation.HeroSlots, int(req.GetSlotPos()))
	return nil
}

// FormationGetPb 获取单个队列（proto）
func FormationGetPb(role *game_roles.Role, formationID uint64) *pb_maps_march.Formation {
	formation := role.GetFormations().GetFormationByID(formationID)
	if formation == nil {
		return nil
	}
	return formatFormation(formation.RoleFormation)
}

// FormationListPb 队列列表（proto）；cityID=0 返回全部
func FormationListPb(role *game_roles.Role, cityID uint64) []*pb_maps_march.Formation {
	list := make([]*pb_maps_march.Formation, 0)
	for _, modelOne := range role.GetFormations().ListByCity(cityID) {
		list = append(list, formatFormation(modelOne))
	}
	return list
}

//-------------------------------

// setHeroSlot 在 slotPos(1起) 位置设置英雄；越界（超过 maxFormationSlots）返回 nil。
// 槽位是固定位置模型：中间空位用 nil 占位，位置不位移。
func setHeroSlot(slots []*pb_maps_march.HeroSlot, slotPos int, heroID uint64, soldierNum uint32) []*pb_maps_march.HeroSlot {
	pos := slotPos - 1
	if pos < 0 || pos >= maxFormationSlots {
		return nil
	}
	for len(slots) <= pos {
		slots = append(slots, nil)
	}
	slots[pos] = &pb_maps_march.HeroSlot{HeroId: heroID, SoldierNum: soldierNum}
	return slots
}

// removeHeroSlotAt 清空 slotPos(1起) 位置的英雄（保持位置不变）
func removeHeroSlotAt(slots []*pb_maps_march.HeroSlot, slotPos int) []*pb_maps_march.HeroSlot {
	pos := slotPos - 1
	if pos >= 0 && pos < len(slots) {
		slots[pos] = nil
	}
	return slots
}

// formatFormation 模型 → proto（保留 nil 空位，前端按位置展示）
func formatFormation(f *game_models.RoleFormation) *pb_maps_march.Formation {
	if f == nil {
		return nil
	}
	return &pb_maps_march.Formation{
		FormationId: f.ID,
		CityId:      f.CityID,
		HeroSlots:   f.HeroSlots,
	}
}
