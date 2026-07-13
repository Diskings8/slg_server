package map_datas

import (
	"sync"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_buildings"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

// MapInfo 地图格子信息，包含格子的坐标、等级、类型、归属服务器以及叠加的建筑和事件
type MapInfo struct {
	rwLock           sync.RWMutex
	mapID            cores_declarations.MapID
	coreMapID        cores_declarations.MapID
	x                int
	y                int
	serverID         uint32
	ownerID          uint64
	Level            cores_declarations.MapLevel
	configID         uint32
	ElementType      cores_declarations.ElementType
	protectedEndTime int64
	overlayEvent     *map_events.OverlayEvent
	overlayBuilding  *map_buildings.OverlayBuilding
}

func (mi *MapInfo) GetMapID() cores_declarations.MapID {
	return mi.mapID
}

func (mi *MapInfo) GetBaseMapID() cores_declarations.MapID {
	return mi.coreMapID
}

func (mi *MapInfo) GetPointX() int {
	return mi.x
}

func (mi *MapInfo) GetPointY() int {
	return mi.y
}

func (mi *MapInfo) GetServerID() uint32 {
	return mi.serverID
}

func (mi *MapInfo) GetLevel() cores_declarations.MapLevel {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.Level
}

func (mi *MapInfo) GetElementID() uint32 {
	return mi.configID
}

func (mi *MapInfo) GetElementType() cores_declarations.ElementType {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ElementType
}

//----------------Lock----------------//

func (mi *MapInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MapInfo) Unlock() {
	mi.rwLock.Unlock()
}

// -------------------
func (mi *MapInfo) Free() {
	// todo
}
