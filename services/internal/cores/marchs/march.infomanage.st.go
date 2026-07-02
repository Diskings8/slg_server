package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
)

type MarchInfoManage struct {
	TickChan             chan *MarchInfo
	MarchTimeType        cores_declarations.MarchTimeType // 行军时间类型
	allMarch             map[cores_declarations.MarchID]*MarchInfo
	allMarchLock         sync.RWMutex
	allAssembleMarch     map[cores_declarations.MarchID][]*MarchInfo
	allAssembleMarchLock sync.RWMutex
	mapConfig            map_datas.MapConfigI
	save                 atomic.Bool
}

func (mm *MarchInfoManage) GetConfig() map_datas.MapConfigI { return mm.mapConfig }

func (mm *MarchInfoManage) addMarchInfo(add *MarchInfo) {
	mm.allMarchLock.Lock()
	defer mm.allMarchLock.Unlock()
	mm.allMarch[add.MarchID] = add
}
