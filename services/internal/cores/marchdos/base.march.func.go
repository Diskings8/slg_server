package marchdos

import (
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

func NewBaseMarch(manage *map_managers.MapManager, marchInfo *marchs.MarchInfo) *BaseMarch {
	_, toMapID, srcMapID := marchInfo.GetMapIDs()
	baseMarch := &BaseMarch{
		marchManage: manage,
		marchInfo:   marchInfo,
	}
	// 获取起源地的信息

}
