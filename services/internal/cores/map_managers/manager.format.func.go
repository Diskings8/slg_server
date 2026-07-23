package map_managers

import (
	"server.slg.com/api/protocol/pb/pb_camera"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_datas/map_buildings"
	"server.slg.com/services/internal/cores/marchs"
)

// FormatMapInfo2Pb 将地图数据格式化为 protobuf 消息
//
// 遍历 sliceInfo 中每个地图格子，填充 pb_camera.MapInfo 的字段。
// LandCover 默认 1，如格子上有玩家主城则设为 HallLandCover(3)，
// 如有 NPC 城市则从 overlay building 中获取联盟 ID 并填充城市数据。
//
// TODO: 随着业务类型增加，此处需补充更多游戏特定类型的格式化逻辑
//   - 资源点/怪物的 ElementType 映射
//   - 联盟 ID 的获取链路（目前依赖 NpcCity 的 CurOccUnionID）
//   - 玩家主城的额外信息（联盟、城墙等）
func (mm *MapManager) FormatMapInfo2Pb(sliceInfo []*map_datas.MapInfo, resp *[]*pb_camera.MapInfo) {
	for _, mapInfo := range sliceInfo {
		pbInfo := MapPBGet()

		pbInfo.MapId = int32(mapInfo.GetMapID())
		pbInfo.ServerId = mapInfo.GetServerID()
		pbInfo.RoleId = mapInfo.GetOwnerID()
		pbInfo.LandCover = 1

		// 检查 overlay building，获取城市和联盟信息
		overlayBuilding := mapInfo.GetOverlayBuilding()
		if overlayBuilding != nil {
			building := overlayBuilding.GetBuilding()
			if building != nil {
				switch b := building.(type) {
				case *map_buildings.NpcCity:
					pbInfo.UnionId = b.CurOccUnionID
					// LandCover 由 NPC 城市配置决定，默认为 HallLandCover
					pbInfo.LandCover = cores_declarations.HallLandCover
					// TODO: 根据 CityData proto 定义填充 pbInfo.City
				}
			}
		}

		// 检查是否为玩家主城（ownerID > 0 且与 coreMapID 一致）
		if mapInfo.GetOwnerID() > 0 && !mapInfo.GetBaseMapID().IsInvalid() {
			pbInfo.LandCover = cores_declarations.HallLandCover
			// TODO: 从 role 数据中补充玩家联盟 ID 和详细信息
		}

		*resp = append(*resp, pbInfo)
	}
}

// FormatMarchInfo2Pb 将行军信息格式化为 protobuf 消息
//
// 注意：调用方需保证 info 在调用期间不被释放；
// 内部已通过 RwLock.RLock 保证并发安全。
func (mm *MapManager) FormatMarchInfo2Pb(info *marchs.MarchInfo) *pb_maps_march.MarchInfo {
	info.RwLock.RLock()
	defer info.RwLock.RUnlock()
	outPB := &pb_maps_march.MarchInfo{
		MarchId:      info.MarchID.Uint64(),
		FromRoleId:   info.GetFromRoleID(),
		ExecRoleId:   info.GetExecRoleID(),
		SrcFromMapId: info.GetSrcFromMapID().Int32(),
		FromMapId:    info.GetFromMapID().Int32(),
		ToMapId:      info.GetToMapID().Int32(),
		State:        info.GetMarchState(),
		StartTime:    info.GetStartTimeUx(),
		EndTime:      info.GetEndTimeUx(),
		UnionId:      info.UnionID,
		TeamInfo:     info.GetTeam().Format2Pb(),
	}
	return outPB
}
