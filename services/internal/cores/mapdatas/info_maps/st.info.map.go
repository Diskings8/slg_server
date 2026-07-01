package info_maps

import (
	"sync"

	"server.slg.com/services/internal/cores/mapdatas/building_maps"
	"server.slg.com/services/internal/cores/mapdatas/event_maps"
	"server.slg.com/services/internal/cores/mapdatas/info_maps/declaration_maps"
	"server.slg.com/services/internal/cores/mapdatas/weather_maps"
)

type MapInfo struct {
	rwLock           sync.RWMutex
	mapID            uint64
	baseMapID        uint64
	x                int
	y                int
	serverID         uint32
	level            declaration_maps.MapLevel
	configID         uint32
	elementType      declaration_maps.ElementType
	protectedEndTime int64
	overlayWeather   *weather_maps.OverlayWeather
	overlayEvent     *event_maps.OverlayEvent
	overlayBuilding  *building_maps.OverlayBuilding
}

func (mi *MapInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MapInfo) Unlock() {
	mi.rwLock.Unlock()
}
