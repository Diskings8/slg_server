package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/common/common_declarations"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ common_declarations.AsyncSaveEntityI = new(MarchInfoManager)

// MarchInfoManager 行军信息管理器，维护所有行军的集合，提供行军添加和按类型组织的能力
type MarchInfoManager struct {
	TickerChan           chan *MarchInfo
	MarchTimeType        cores_declarations.MarchTimeType // 行军时间类型
	allMarch             map[cores_declarations.MarchID]*MarchInfo
	allMarchLock         sync.RWMutex
	allAssembleMarch     map[cores_declarations.MarchID][]*MarchInfo
	allAssembleMarchLock sync.RWMutex
	mapConfig            cores_declarations.MapConfigI
	mapMarch             []MapAttribute
	tableName            string
	saving               atomic.Bool
}

func (mm *MarchInfoManager) GetConfig() cores_declarations.MapConfigI { return mm.mapConfig }

func (mm *MarchInfoManager) addMarchInfo(add *MarchInfo) {
	mm.allMarchLock.Lock()
	defer mm.allMarchLock.Unlock()
	mm.allMarch[add.MarchID] = add
	// 放入地块属性
	mm.TickerChan <- add
}

func (mm *MarchInfoManager) GetTableName() string {
	return mm.tableName
}
