package map_datas

import (
	"errors"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
)

func (mdm *MapDataManager) Clear(mapIDs []cores_declarations.MapID) {
	// todo
	panic("implement me")
}

func (mdm *MapDataManager) SetRoleMainCity(roleCityState cores_declarations.RoleMainCityState, dataSlice []*MapInfo, roleBrief *pb_role.RoleBrief) error {
	var coreIndex int
	switch roleCityState {
	case cores_declarations.RoleMainCityStateNormal:
		if len(dataSlice) != cores_declarations.RoleMainCityStateNormalCoverCount {
			return errors.New("地块数量不对")
		}
		coreIndex = cores_declarations.Land1CoverBaseKey
	default:
		if len(dataSlice) != cores_declarations.RoleMainCityStatePortableCoverCount {
			return errors.New("地块数量不对")
		}
		coreIndex = cores_declarations.Land3CoverBaseKey
	}
	// 检测位置可使用情况 todo

	//
	coreMapInfo := dataSlice[coreIndex]
	for _, mapInfo := range dataSlice {
		mapInfo.Free()
		mapInfo.serverID = roleBrief.GetRoleBaseInfo().GetServerId()
		mapInfo.ownerID = roleBrief.GetRoleBaseInfo().GetId()
		mapInfo.coreMapID = coreMapInfo.mapID

		// aoi 更新
	}
	mdm.Save(dataSlice...)

	// 更新角色数据 todo

	// aoi更新

	return nil
}
