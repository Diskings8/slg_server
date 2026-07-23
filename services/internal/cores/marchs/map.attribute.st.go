package marchs

import (
	"sync"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// MapAttribute 地图行军属性，管理地图上的驻守队伍和经过的行军集合
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
	ma.marchMap.Range(func(cores_declarations.MarchID, *MarchInfo) bool {
		l++
		return true
	})
	return
}

// GetMapMarchLen 获取地块上行军总数
func (ma *MapAttribute) GetMapMarchLen() int {
	if ma == nil {
		return 0
	}
	return ma.marchMap.Len()
}

// GetMarchIDList 获取地块上所有行军 ID 列表
func (ma *MapAttribute) GetMarchIDList() []cores_declarations.MarchID {
	if ma == nil {
		return nil
	}
	out := make([]cores_declarations.MarchID, 0, ma.GetMapMarchLen())
	ma.marchMap.Range(func(id cores_declarations.MarchID, _ *MarchInfo) bool {
		out = append(out, id)
		return true
	})
	return out
}

// CleanAllMarch 清除地块上所有行军（测试用）
func (ma *MapAttribute) CleanAllMarch() {
	if ma == nil {
		return
	}
	for _, id := range ma.marchMap.Keys() {
		ma.marchMap.Delete(id)
	}
	ma.assistLocker.Lock()
	ma.assistSlice = ma.assistSlice[:0]
	ma.assistLocker.Unlock()
}
