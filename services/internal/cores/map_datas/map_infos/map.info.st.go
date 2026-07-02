package map_infos

import (
	"sync"

	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_datas/map_buildings"
	"server.slg.com/services/internal/cores/map_datas/map_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

var _ map_datas.MapInfoI = (*MapInfo)(nil)

type MapInfo struct {
	rwLock           sync.RWMutex
	mapID            uint64
	baseMapID        uint64
	x                int
	y                int
	serverID         uint32
	level            map_declarations.MapLevel
	configID         uint32
	elementType      map_declarations.ElementType
	protectedEndTime int64
	overlayEvent     *map_events.OverlayEvent
	overlayBuilding  *map_buildings.OverlayBuilding
}

func (mi *MapInfo) MapID() uint64 {
	return mi.mapID
}

func (mi *MapInfo) BaseMapID() uint64 {
	return mi.baseMapID
}

func (mi *MapInfo) PointX() int {
	return mi.x
}

func (mi *MapInfo) PointY() int {
	return mi.y
}

func (mi *MapInfo) ServerID() uint32 {
	return mi.serverID
}

func (mi *MapInfo) Level() map_declarations.MapLevel {
	return mi.level
}

func (mi *MapInfo) ElementID() uint32 {
	return mi.configID
}

//----------------Lock----------------//

func (mi *MapInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MapInfo) Unlock() {
	mi.rwLock.Unlock()
}
