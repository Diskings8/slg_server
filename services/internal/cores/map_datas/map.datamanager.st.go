package map_datas

import (
	"sync/atomic"

	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_aois"
)

// MapDataManager 地图数据管理器，负责地图格子的初始化、保存和 AOI 管理
type MapDataManager struct {
	Id        uint64
	waitSave  hashmaps.Map[cores_declarations.MapID, *MapInfo]
	config    cores_declarations.MapConfigI
	tableName string
	saving    atomic.Bool

	AOI *map_aois.ScreenData
	// 出生块
	BornAts cores_declarations.BornBlockI

	mapData []MapInfo
}

func (mdm *MapDataManager) GetConfig() cores_declarations.MapConfigI {
	return mdm.config
}

func (mdm *MapDataManager) Init(mapD []MapInfo) {
	mdm.mapData = mapD
	for mapID := int32(0); mapID < mdm.config.MapCount(); mapID++ {
		v := &mdm.mapData[mapID]
		if v.GetMapID() == cores_declarations.InvalidMapID {
			continue
		}
		// 特殊地块需要处理

		if v.GetBaseMapID() == v.GetMapID() {
			mdm.AOI.MapDataAdd(cores_declarations.MapID(mapID))
		}
	}
}

func (mdm *MapDataManager) Tag() string {
	return "MapDataManager"
}

func (mdm *MapDataManager) GetMapInfo(mapID cores_declarations.MapID) (*MapInfo, bool) {
	if mapID < 0 || mapID >= cores_declarations.MapID(len(mdm.mapData)) {
		return nil, false
	}
	d := &mdm.mapData[mapID]
	if d.mapID == cores_declarations.InvalidMapID {
		return nil, false
	}
	return d, true
}

func (mdm *MapDataManager) GetMapInfoSlice(mapIDs []cores_declarations.MapID) []*MapInfo {
	result := make([]*MapInfo, 0, len(mdm.mapData))
	for _, mapID := range mapIDs {
		if d, ok := mdm.GetMapInfo(mapID); ok {
			result = append(result, d)
		}
	}
	return result
}

func (mdm *MapDataManager) Range(f func(m *MapInfo) bool) {
	for index := range mdm.mapData {
		tmp := &mdm.mapData[index]
		if tmp.GetMapID() == cores_declarations.InvalidMapID {
			continue
		}
		if !f(tmp) {
			return
		}
	}
}

func (mdm *MapDataManager) TryLock(mapList []*MapInfo) bool {
	locked := make([]*MapInfo, 0, len(mapList))
	for _, mapD := range mapList {
		if !mapD.TryLock() {
			goto failUnlock
		}
		locked = append(locked, mapD)
	}
	return true
failUnlock:
	for _, mapD := range locked {
		mapD.Unlock()
	}
	return false
}

func (mdm *MapDataManager) Unlock(mapList []*MapInfo) {
	exMapID := make(map[cores_declarations.MapID]bool, len(mapList))
	for _, mapD := range mapList {
		if _, ok := exMapID[mapD.GetMapID()]; ok {
			continue
		}
		mapD.Unlock()
		exMapID[mapD.GetMapID()] = true
	}
}

func (mdm *MapDataManager) Save(list ...*MapInfo) {
	for _, m := range list {
		mdm.waitSave.Store(m.GetMapID(), m)
	}
	// todo save
}

func (mdm *MapDataManager) SaveDo() {

}

func (mdm *MapDataManager) SetHall(data []*MapInfo, brief *pb_role.RoleBrief) error {
	panic("implement me")
	return nil
}

type LockMapSlice struct {
	data []*MapInfo
	mdm  *MapDataManager
}

// Unlock 解锁
func (l LockMapSlice) Unlock() {
	if l.mdm == nil {
		return
	}
	l.mdm.Unlock(l.data)
}

// Data 数据
func (l LockMapSlice) Data() []*MapInfo {
	if l.mdm == nil {
		return nil
	}
	return l.data
}
