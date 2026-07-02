package marchs

import (
	"sync"
	"sync/atomic"

	"server.slg.com/services/internal/cores/declaration_cores"
	"server.slg.com/services/internal/cores/mapdatas/info_maps/interface_maps"
)

type MarchInfoManage struct {
	TickChan             chan *MarchInfo
	MarchTimeType        declaration_cores.MarchTimeType // 行军时间类型
	allMarch             map[declaration_cores.MarchID]*MarchInfo
	allMarchLock         sync.RWMutex
	allAssembleMarch     map[declaration_cores.MarchID]*MarchInfo
	allAssembleMarchLock sync.RWMutex
	mapConfig            interface_maps.MapConfigI
	save                 atomic.Bool
}

func (mm *MarchInfoManage) GetConfig() interface_maps.MapConfigI { return mm.mapConfig }

func (mm *MarchInfoManage) addMarchInfo(add *MarchInfo) {
	mm.allMarchLock.Lock()
	defer mm.allMarchLock.Unlock()
	mm.allMarch[add.MarchID] = add
}
