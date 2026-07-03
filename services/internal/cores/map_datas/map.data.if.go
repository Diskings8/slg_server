package map_datas

import "server.slg.com/services/internal/cores/cores_declarations"

type MapConfigI interface {
	// MapCount 地图总数
	MapCount() uint32

	MapID2XY(id cores_declarations.MapID) (x, y int32)
}
