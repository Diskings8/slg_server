package map_managers

import "server.slg.com/services/internal/cores/cores_declarations"

func (mm *MapManager) TickerAddMarch(marchID cores_declarations.MarchID, marchEndTime int64) {
	mm.timeMarchLock.Lock()
	defer mm.timeMarchLock.Unlock()
	if _, ok := mm.timeMarch[marchEndTime]; !ok {
		mm.timeMarch[marchEndTime] = make(map[cores_declarations.MarchID]struct{})
	}
	mm.timeMarch[marchEndTime][marchID] = struct{}{}
}

// TickerAddMap TickerAddMap
func (mm *MapManager) TickerAddMap(mapID cores_declarations.MapID, clearTime int64) {
	mm.timeMapLock.Lock()
	defer mm.timeMapLock.Unlock()
	if _, ok := mm.timeMap[clearTime]; !ok {
		mm.timeMap[clearTime] = make(map[cores_declarations.MapID]struct{})
	}
	mm.timeMap[clearTime][mapID] = struct{}{}
}

// TickerAddMapList TickerAddMapList
func (mm *MapManager) TickerAddMapList(mapIDList []cores_declarations.MapID, clearTime int64) {
	mm.timeMapLock.Lock()
	defer mm.timeMapLock.Unlock()
	if _, ok := mm.timeMap[clearTime]; !ok {
		mm.timeMap[clearTime] = make(map[cores_declarations.MapID]struct{})
	}
	for _, mapID := range mapIDList {
		mm.timeMap[clearTime][mapID] = struct{}{}
	}
}
