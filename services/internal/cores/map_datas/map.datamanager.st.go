package map_datas

import (
	"sync/atomic"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/aois"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type MapDataManager struct {
	Id        uint64
	waitSave  hashmaps.Map[cores_declarations.MapID, MapInfoI]
	config    MapConfigI
	tableName string
	saving    atomic.Bool

	AOI     *aois.ScreenData
	mapData []MapInfoI
}

func (mdm *MapDataManager) GetConfig() MapConfigI {
	return mdm.config
}

func (mdm *MapDataManager) Init(mapD []MapInfoI) {
	mdm.mapData = mapD
}
