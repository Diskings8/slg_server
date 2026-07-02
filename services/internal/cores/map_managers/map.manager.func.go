package map_managers

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

func NewMapManager(
	roomID uint64,
	mapGroup cores_declarations.MapGroup,
	mapsData map_datas.MapInfoI,
	marchManage *marchs.MarchInfoManage,

) *MapManager {
	mm := &MapManager{}
	return mm
}
