package map_infos

import (
	"sync"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_buildings"
	"server.slg.com/services/internal/cores/map_datas/map_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

// MapInfo 地图格子信息，包含格子的坐标、等级、类型、归属服务器以及叠加的建筑和事件
type MapInfo struct {
	rwLock           sync.RWMutex
	mapID            cores_declarations.MapID
	baseMapID        cores_declarations.MapID
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

func (mi *MapInfo) MapID() cores_declarations.MapID {
	return mi.mapID
}

func (mi *MapInfo) BaseMapID() cores_declarations.MapID {
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
