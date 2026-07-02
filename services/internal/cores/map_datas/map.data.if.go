package map_datas

import (
	"server.slg.com/services/internal/cores/map_datas/map_declarations"
)

type MapInfoI interface {
	MapID() uint64
	BaseMapID() uint64
	PointX() int
	PointY() int
	ServerID() uint32
	Level() map_declarations.MapLevel
	ElementID() uint32
}

type MapConfigI interface {
	// MapCount 地图总数
	MapCount() int32
}
