package map_datas

import (
	"errors"
	"sync/atomic"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/aois"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_infos"
)

// MapDataManager 地图数据管理器，负责地图格子的初始化、保存和 AOI 管理
type MapDataManager struct {
	Id        uint64
	waitSave  hashmaps.Map[cores_declarations.MapID, *map_infos.MapInfo]
	config    MapConfigI
	tableName string
	saving    atomic.Bool

	AOI     *aois.ScreenData
	mapData []*map_infos.MapInfo
}

func (mdm *MapDataManager) GetConfig() MapConfigI {
	return mdm.config
}

func (mdm *MapDataManager) Init(mapD []*map_infos.MapInfo) {
	mdm.mapData = mapD
}

func (mdm *MapDataManager) GetMapInfo(mapID cores_declarations.MapID) (*map_infos.MapInfo, error) {
	for _, v := range mdm.mapData {
		if v.MapID() == mapID {
			return v, nil
		}
	}
	return nil, errors.New("map not found")
}
