package map_managers

import (
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// CreateRole 创建角色位置
func (mm *MapManager) CreateRole(roleBrief *pb_role.RoleBrief) ([]cores_declarations.MapID, error) {
	mapIDs, lockMapSlice, _, baseMapID, freeBornFunc, err := mm.GetMapDataManager().GetFreeBorn()
	if err != nil {
		return nil, err
	}

	defer lockMapSlice.Unlock()
	err = mm.mapDataManager.SetHall(lockMapSlice.Data(), roleBrief)
	if err != nil {
		freeBornFunc()
		return nil, err
	}

	mm.UpdateMapPush(baseMapID)

	return mapIDs, err
}
