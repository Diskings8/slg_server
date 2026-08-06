package map_managers

import (
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// CreateRole 创建角色位置，返回主城核心 MapID
func (mm *MapManager) CreateRole(roleBrief *pb_role.RoleBrief) (cores_declarations.MapID, error) {
	_, lockMapSlice, _, coreMapID, freeBornFunc, err := mm.GetMapDataManager().GetFreeBorn()
	if err != nil {
		return cores_declarations.InvalidMapID, err
	}

	defer lockMapSlice.Unlock()
	err = mm.mapDataManager.SetRoleMainCity(cores_declarations.RoleMainCityStateNormal, lockMapSlice.Data(), roleBrief)
	if err != nil {
		freeBornFunc()
		return cores_declarations.InvalidMapID, err
	}

	mm.UpdateMapPush(coreMapID)

	return coreMapID, nil
}
