package marchs

import (
	"sync"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type MapAttribute struct {
	assistSlice  []*MarchInfo // 驻守队伍
	assistLocker sync.RWMutex
	marchMap     hashmaps.Map[cores_declarations.MarchID, *MarchInfo] // 地图管理行军
}

func (ma *MapAttribute) Init() {}

func (ma *MapAttribute) marchAdd(mi *MarchInfo) {
	ma.marchMap.Store(mi.MarchID, mi)
}

func (ma *MapAttribute) marchDel(marchID cores_declarations.MarchID) {
	ma.marchMap.Delete(marchID)
}

func (ma *MapAttribute) GetMapMarch(container map[cores_declarations.MarchID]*MarchInfo) map[cores_declarations.MarchID]*MarchInfo {
	if ma != nil {
		ma.marchMap.Range(func(id cores_declarations.MarchID, info *MarchInfo) bool {
			container[id] = info
			return true
		})
	}
	return container
}

func (ma *MapAttribute) RangeMapMarch(f func(id cores_declarations.MarchID, info *MarchInfo) bool) {
	if ma != nil {
		ma.marchMap.Range(f)
	}
}

func (ma *MapAttribute) GetAllMapMarchLen() (l int) {
	if ma == nil {
		return
	}
	ma.RangeMapMarch(func(cores_declarations.MarchID, *MarchInfo) bool {
		l++
		return true
	})
	return
}
