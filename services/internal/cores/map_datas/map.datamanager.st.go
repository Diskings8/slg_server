package map_datas

import (
	"sync/atomic"

	"go.uber.org/zap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/asyncsave_entity"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_aois"
)

const (
	maxSaveLen = 1000
)

var _ common_declarations.AsyncSaveEntityI = new(MapDataManager)

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
		mapD.UnLock()
	}
	return false
}

func (mdm *MapDataManager) UnLock(mapList []*MapInfo) {
	exMapID := make(map[cores_declarations.MapID]bool, len(mapList))
	for _, mapD := range mapList {
		if _, ok := exMapID[mapD.GetMapID()]; ok {
			continue
		}
		mapD.UnLock()
		exMapID[mapD.GetMapID()] = true
	}
}

func (mdm *MapDataManager) IsDelete() bool {
	return false
}

func (mdm *MapDataManager) Saving() bool {
	return mdm.saving.Load()
}

func (mdm *MapDataManager) Save(list ...*MapInfo) {
	for _, m := range list {
		mdm.waitSave.Store(m.GetMapID(), m)
	}
	asyncsave_entity.EntitySaveFunc(mdm)
}

// save 批量保存
func (mdm *MapDataManager) save(db common_declarations.DbcI, waitSlice [maxSaveLen]*MapInfo, num int) {
	tmp := waitSlice[:num]
	err := db.Table(mdm.tableName).Save(tmp).Error()
	if err != nil {
		loggers.Logger.Error("save map data failed", zap.Error(err))
	}

	for _, v := range tmp {
		if err == nil {
			mdm.waitSave.Delete(v.GetMapID())
		}
		v.UnLock() // 外层上锁
	}
}

func (mdm *MapDataManager) SaveDo() {
	dbWriter := dbconn.GetWriteDbConn()
	num := 0
	waitSlice := [maxSaveLen]*MapInfo{}
	mdm.waitSave.Range(func(_ cores_declarations.MapID, v *MapInfo) bool {
		if v.TryLock() {
			waitSlice[num] = v
			num++

			if num >= maxSaveLen {
				mdm.save(dbWriter, waitSlice, num)
				num = 0
			}
		}
		return true
	})
	if num > 0 {
		mdm.save(dbWriter, waitSlice, num)
	}
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
	l.mdm.UnLock(l.data)
}

// Data 数据
func (l LockMapSlice) Data() []*MapInfo {
	if l.mdm == nil {
		return nil
	}
	return l.data
}
