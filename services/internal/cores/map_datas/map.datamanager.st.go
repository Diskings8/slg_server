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
//
// 存储策略：
//   - bigMapData（稠密）：值类型 []MapInfo flat slice，O(1) 索引，适合大地图
//   - smallMapData（稀疏）：map[MapID]*MapInfo，仅存有值格子，适合事件/战场小地图
//   - isSparse 为 true 时使用 smallMapData，否则使用 bigMapData
type MapDataManager struct {
	Id        uint64
	waitSave  hashmaps.Map[cores_declarations.MapID, *MapInfo]
	config    cores_declarations.MapConfigI
	tableName string
	saving    atomic.Bool

	AOI     *map_aois.ScreenData
	BornAts cores_declarations.BornBlockI

	isSparse     bool                                    // true → smallMapData，false → bigMapData
	bigMapData   []MapInfo                               // 稠密：值类型 slice，O(1) 索引
	smallMapData map[cores_declarations.MapID]*MapInfo   // 稀疏：仅存有值格子
}

// Init 初始化，isSparse 控制使用哪种存储策略
func (mdm *MapDataManager) Init(mapD []MapInfo, isSparse ...bool) {
	if len(isSparse) > 0 && isSparse[0] {
		mdm.isSparse = true
		mdm.smallMapData = make(map[cores_declarations.MapID]*MapInfo, len(mapD))
		for i := range mapD {
			v := &mapD[i]
			if v.GetMapID() != cores_declarations.InvalidMapID {
				mdm.smallMapData[v.GetMapID()] = v
				if v.GetBaseMapID() == v.GetMapID() {
					mdm.AOI.MapDataAdd(v.GetMapID())
				}
			}
		}
		return
	}

	mdm.isSparse = false
	mdm.bigMapData = mapD
	for mapID := int32(0); mapID < mdm.config.MapCount(); mapID++ {
		v := &mdm.bigMapData[mapID]
		if v.GetMapID() == cores_declarations.InvalidMapID {
			continue
		}
		if v.GetBaseMapID() == v.GetMapID() {
			mdm.AOI.MapDataAdd(cores_declarations.MapID(mapID))
		}
	}
}

func (mdm *MapDataManager) Tag() string {
	return "MapDataManager"
}

func (mdm *MapDataManager) GetConfig() cores_declarations.MapConfigI {
	return mdm.config
}

// GetMapInfo 获取地图格子数据，根据存储策略路由
func (mdm *MapDataManager) GetMapInfo(mapID cores_declarations.MapID) (*MapInfo, bool) {
	if mdm.isSparse {
		d, ok := mdm.smallMapData[mapID]
		return d, ok
	}

	id := int32(mapID)
	if id < 0 || id >= int32(len(mdm.bigMapData)) {
		return nil, false
	}
	d := &mdm.bigMapData[id]
	if d.mapID == cores_declarations.InvalidMapID {
		return nil, false
	}
	return d, true
}

func (mdm *MapDataManager) GetMapInfoSlice(mapIDs []cores_declarations.MapID) []*MapInfo {
	if mdm.isSparse {
		result := make([]*MapInfo, 0, len(mapIDs))
		for _, mapID := range mapIDs {
			if d, ok := mdm.GetMapInfo(mapID); ok {
				result = append(result, d)
			}
		}
		return result
	}

	result := make([]*MapInfo, 0, len(mdm.bigMapData))
	for _, mapID := range mapIDs {
		if d, ok := mdm.GetMapInfo(mapID); ok {
			result = append(result, d)
		}
	}
	return result
}

func (mdm *MapDataManager) Range(f func(m *MapInfo) bool) {
	if mdm.isSparse {
		for _, v := range mdm.smallMapData {
			if v.GetMapID() == cores_declarations.InvalidMapID {
				continue
			}
			if !f(v) {
				return
			}
		}
		return
	}

	for index := range mdm.bigMapData {
		tmp := &mdm.bigMapData[index]
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

func (mdm *MapDataManager) Saving() *atomic.Bool {
	return &mdm.saving
}

func (mdm *MapDataManager) Save(list ...*MapInfo) {
	for _, m := range list {
		mdm.waitSave.Store(m.GetMapID(), m)
	}
	asyncsave_entity.EntitySaveFunc(mdm)
}

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
		v.UnLock()
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

// IsSparse 返回当前是否使用稀疏存储
func (mdm *MapDataManager) IsSparse() bool {
	return mdm.isSparse
}

type LockMapSlice struct {
	data []*MapInfo
	mdm  *MapDataManager
}

func (l LockMapSlice) Unlock() {
	if l.mdm == nil {
		return
	}
	l.mdm.UnLock(l.data)
}

func (l LockMapSlice) Data() []*MapInfo {
	if l.mdm == nil {
		return nil
	}
	return l.data
}
