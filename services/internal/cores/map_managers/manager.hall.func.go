package map_managers

import "server.slg.com/api/protocol/pb/pb_role"

// CreateRole 创建角色位置
func (mm *MapManager) CreateRole(roleBrief *pb_role.RoleBrief) ([]int32, error) {
	mapIDs, lockMapSlice, _, baseMapID, freeBornFunc, err := mm.GetMapDataManager().GetFreeBorn(areaLevel)
	if err != nil {
		return nil, err
	}

	defer lockMapSlice.Unlock()
	err = m.Map().SetHall(lockMapSlice.Data(), roleBrief)
	if err != nil {
		freeBornFunc()
		return nil, err
	}

	m.UpMap(baseMapID)

	return mapIDs, err
}
