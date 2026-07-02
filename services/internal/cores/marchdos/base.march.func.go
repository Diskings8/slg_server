package marchdos

import (
	"go.uber.org/zap"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

func NewBaseMarch(mapManage *map_managers.MapManager, marchInfo *marchs.MarchInfo) *BaseMarch {
	_, toMapID, srcMapID := marchInfo.GetMapIDs()
	baseMarch := &BaseMarch{
		marchManage: mapManage.GetMarchManage(),
		marchInfo:   marchInfo,
	}
	// 获取起源地的信息
	srcFromMapInfo, err := mapManage.GetMapDataManager().GetMapInfo(srcMapID)
	if err != nil {
		loggers.Logger.Error("GetMapInfo", zap.Error(err))
	}
	baseMarch.fromMapInfo = srcFromMapInfo
	// 获取目标地的信息
	toMapInfo, err := mapManage.GetMapDataManager().GetMapInfo(toMapID)
	if err != nil {
		loggers.Logger.Error("GetMapInfo", zap.Error(err))
	}
	baseMarch.toMapInfo = toMapInfo

	return baseMarch
}
