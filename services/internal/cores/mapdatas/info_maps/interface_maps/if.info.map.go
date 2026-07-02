package interface_maps

import "server.slg.com/services/internal/cores/mapdatas/info_maps/declaration_maps"

type MapInfoI interface {
	MapID() uint64
	BaseMapID() uint64
	PointX() int
	PointY() int
	ServerID() uint32
	Level() declaration_maps.MapLevel
	ElementID() uint32
}

type MapConfigI interface {
	// MapCount 地图总数
	MapCount() int32
}
