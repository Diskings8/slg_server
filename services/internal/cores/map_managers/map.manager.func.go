package map_managers

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_blocks"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

func NewMapManager(
	roomID uint64,
	mapGroup cores_declarations.MapGroup,
	mapDataManager *map_datas.MapDataManager,
	marchManage *marchs.MarchInfoManager,
	mapBlock *map_blocks.MapBlock,
) *MapManager {
	mm := &MapManager{
		RoomID:          roomID,
		MapGroup:        mapGroup,
		mapDataManager:  mapDataManager,
		marchManage:     marchManage,
		mapBlock:        mapBlock,
		timeMarch:       make(map[int64]map[cores_declarations.MarchID]struct{}),
		timeMap:         make(map[int64]map[cores_declarations.MapID]struct{}),
		waitUpdateMapID: make(map[cores_declarations.MapID]struct{}),
	}
	return mm
}
