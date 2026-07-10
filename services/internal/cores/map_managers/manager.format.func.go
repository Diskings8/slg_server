package map_managers

import (
	"server.slg.com/api/protocol/pb/pb_camera"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

func (mm *MapManager) FormatMapInfo2Pb(sliceInfo []*map_datas.MapInfo, resp *[]*pb_camera.MapInfo) {
	// 后续需要预处理返回的数据的时候可以在这里处理 todo
	return
}

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
