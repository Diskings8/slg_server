package map_managers

import (
	"slices"

	"server.slg.com/services/internal/cores/aois"
	"server.slg.com/services/internal/cores/marchs"
)

// MarchAOISetupSingle 行军AOI设置
func (mm *MapManager) MarchAOISetupSingle(marchInfo *marchs.MarchInfo) {
	fromMapX, fromMapY := mm.GetConf().MapID2XY(marchInfo.GetFromMapID())
	toMapX, toMapY := mm.GetConf().MapID2XY(marchInfo.GetToMapID())
	mm.MarchAOISetup(marchInfo, fromMapX, fromMapY, toMapX, toMapY)
}

// MarchAOISetup 行军AOI设置
func (mm *MapManager) MarchAOISetup(marchInfo *marchs.MarchInfo, startX, startY, endX, endY int32) {
	if marchInfo.IsVirtual() {
		marchInfo.RwLock.RLock()
		path := slices.Clone(marchInfo.Path)
		marchInfo.RwLock.RUnlock()

		for _, mapID := range path {
			mm.GetMapDataManager().AOI.GetScreen(mapID).MarchAdd(marchInfo)
			break
		}
	} else {
		screenList := mm.GetMapDataManager().AOI.MovePath(startX, startY, endX, endY, &([]*aois.Screen[int32]{}))
		screenLen := len(screenList)
		for index, v := range screenList {
			if index == 0 || index == screenLen-1 {
				v.MarchAdd(marchInfo)
				continue
			}
			v.CrossMarchAdd(marchInfo)
		}
	}
}
