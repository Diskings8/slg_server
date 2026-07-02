package map_managers

import (
	"sync"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_blocks"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

type MapManager struct {
	RoomID        uint64
	MapGroup      cores_declarations.MapGroup
	maps          map_datas.MapInfoI
	marchManage   *marchs.MarchInfoManage
	timeMarch     map[int64]map[cores_declarations.MarchID]struct{}
	timeMarchLock sync.Mutex
	timeMap       map[int64]map[cores_declarations.MapID]struct{}
	timeMapLock   sync.Mutex
	marchDoFunc   func(id cores_declarations.MarchID, manager *MapManager)
	mapBlock      *map_blocks.MapBlock

	//
	waitUpdateMapID   map[cores_declarations.MapID]struct{}
	waitUpdateMapLock sync.Mutex
}

func (mm *MapManager) Map() map_datas.MapInfoI {
	return mm.maps
}

func (mm *MapManager) MarchManage() *marchs.MarchInfoManage {
	return mm.marchManage
}

func (mm *MapManager) Block() *map_blocks.MapBlock {
	return mm.mapBlock
}

func (mm *MapManager) GetMapConfig() map_datas.MapConfigI {
	return mm.maps.GetConf()
}
