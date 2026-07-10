package map_managers

import (
	"server.slg.com/api/protocol/pb/pb_camera"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/utils/s2s"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_aois"
	"server.slg.com/services/internal/cores/marchs"
)

func (mm *MapManager) upMapAsync() {
	// 检查需要更新的地图数据
	mm.waitUpdateMapLock.Lock()
	showMapIDSlice := make([]cores_declarations.MapID, 0, len(mm.waitUpdateMapID))
	for mapID := range mm.waitUpdateMapID {
		showMapIDSlice = append(showMapIDSlice, mapID)
	}
	clear(mm.waitUpdateMapID)
	mm.waitUpdateMapLock.Unlock()
	if len(showMapIDSlice) < 1 {
		return
	}

	// 检查需要更新的地图数据的玩家是否存在
	roleMapIDs := make(map[uint64][]cores_declarations.MapID)            // 角色视野内要推送的地图ID
	roleIDConnect := make(map[uint64]cores_declarations.MapRoleConnectI) // 角色ID/连接数据(仅需要推送的)
	for _, mapID := range showMapIDSlice {
		for _, aoiConn := range mm.GetMapDataManager().AOI.AroundConnects(mapID, map[uint64]cores_declarations.MapRoleConnectI{}) {
			roleMapIDs[aoiConn.GetRoleID()] = append(roleMapIDs[aoiConn.GetRoleID()], mapID)
			roleIDConnect[aoiConn.GetRoleID()] = aoiConn
		}
	}
	if len(roleMapIDs) < 1 {
		return
	}

	mapListPB := make([]*pb_camera.MapInfo, 0, len(showMapIDSlice))
	mm.FormatMapInfo2Pb(mm.GetMapDataManager().GetMapInfoSlice(showMapIDSlice), &mapListPB)

	mapIDInfo := make(map[cores_declarations.MapID]*pb_camera.MapInfo, len(mapListPB))
	for _, mapInfo := range mapListPB {
		mapIDInfo[cores_declarations.MapID(mapInfo.MapId)] = mapInfo
	}

	// 整理需要推送给前端的东西
	for roleID, mapIDsTmp := range roleMapIDs {

		rolePush := &pb_camera.PushMapInfo{}
		for _, mapID := range mapIDsTmp {
			mapInfoTmp, ok := mapIDInfo[mapID]
			if !ok {
				continue
			}
			rolePush.MapInfos = append(rolePush.MapInfos, mapInfoTmp)
		}
		mm.roleConnectManager.PushToRoleID(pb_protocol.MsgID_PushMapInfo, rolePush, roleID)
	}
}

// UpdateMapPush 部分地块更新信息需要下推
func (mm *MapManager) UpdateMapPush(mapIDs ...cores_declarations.MapID) {
	if len(mapIDs) <= 0 {
		return
	}

	mm.waitUpdateMapLock.Lock()
	defer mm.waitUpdateMapLock.Unlock()
	for _, mapID := range mapIDs {
		mm.waitUpdateMapID[mapID] = struct{}{}
	}
}

func (mm *MapManager) upMarchSync(marchFormMapID, marchToMapID cores_declarations.MapID, marchPB *pb_maps_march.MarchInfo, receiver ...uint64) {
	pushRoleID := map[uint64]struct{}{}
	for _, v := range receiver {
		pushRoleID[v] = struct{}{}
	}

	startX, startY := mm.GetConf().MapID2XY(marchFormMapID)
	endX, endY := mm.GetConf().MapID2XY(marchToMapID)
	for _, screenData := range mm.GetMapDataManager().AOI.MovePath(startX, startY, endX, endY, &([]*map_aois.Screen[cores_declarations.ScreenID]{})) {
		for roleID := range screenData.Connects(map[uint64]cores_declarations.MapRoleConnectI{}) {
			pushRoleID[roleID] = struct{}{}
		}
	}

	pushRoleIDsSlice := s2s.MapKey2Slice(pushRoleID)

	mm.roleConnectManager.PushToRoleIDs(pb_protocol.MsgID_PushMarchInfo, marchPB, pushRoleIDsSlice...)
}

// UpdateMarchPush 更新的行军信息推送
func (mm *MapManager) UpdateMarchPush(marchInfo *marchs.MarchInfo) {
	marchInfo.RwLock.RLock()
	var (
		marchFromRoleID = marchInfo.FromRoleID
		marchExecRoleID = marchInfo.ExecRoleID
		marchFromMapID  = marchInfo.FromMapID
		marchToMapID    = marchInfo.ToMapID
	)
	marchInfo.RwLock.RUnlock()
	marchPB := mm.FormatMarchInfo2Pb(marchInfo)

	mm.upMarchSync(marchFromMapID, marchToMapID, marchPB, marchFromRoleID, marchExecRoleID)
}
