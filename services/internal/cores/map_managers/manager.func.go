package map_managers

import (
	"time"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_blocks"
	"server.slg.com/services/internal/cores/map_borns"
	"server.slg.com/services/internal/cores/map_connects"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

func NewMapManager(
	roomID uint64,
	mapGroup cores_declarations.MapGroup,
	mapDataManager *map_datas.MapDataManager,
	marchManage *marchs.MarchInfoManager,
	marchDoFunc func(*MapManager, cores_declarations.MarchID),
	marchDoHandleFunc func(*MapManager, *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI,
	op ...Option,
) *MapManager {
	mm := &MapManager{
		RoomID:             roomID,
		MapGroup:           mapGroup,
		mapDataManager:     mapDataManager,
		marchManage:        marchManage,
		roleConnectManager: map_connects.NewRoleConnectManager(mapDataManager.AOI),
		mapBlock:           map_blocks.NewMapBlock(mapDataManager),
		marchDoFunc:        marchDoFunc,
		marchDoFuncHandle:  marchDoHandleFunc,
		timeMarch:          make(map[int64]map[cores_declarations.MarchID]struct{}),
		timeMap:            make(map[int64]map[cores_declarations.MapID]struct{}),
		waitUpdateMapID:    make(map[cores_declarations.MapID]struct{}),
		opts:               evaluateOptions(op),
	}

	initBornManager(mm, mapDataManager)

	return mm
}

// initBornManager 接线地图出生块管理器
//
// 每个地图区块 Store 若干"可出生候选种子"（区块内部偏移点，过滤掉不可出生地形）。
// 出生点分配（GetFreeBorn）以种子为中心展开 3×3 校验完整空地后占为己用。
func initBornManager(mm *MapManager, mapDataManager *map_datas.MapDataManager) {
	bornMgr := map_borns.NewBigMapBornBlockManager(cores_declarations.ServerMapBlockCutNum)
	blockLength := mm.mapBlock.BlockLength()
	// 区块内部偏移点：取区块边长 1/4、1/2、3/4 三档位置，
	// 远离地图边界（保证 3×3 邻域完整），且分散覆盖区块
	offsets := []int32{blockLength / 4, blockLength / 2, blockLength * 3 / 4}
	for i := int32(1); i <= cores_declarations.ServerMapBlockCutNum; i++ {
		sx, sy := mm.mapBlock.FirstXY(cores_declarations.BornBlockID(i))
		seeds := make(map[int32]struct{})
		for _, ox := range offsets {
			for _, oy := range offsets {
				mid := mapDataManager.GetConfig().XY2MapID(sx+ox, sy+oy)
				if info, ok := mapDataManager.GetMapInfo(mid); ok && !info.GetElementType().IsCantBornUse() {
					seeds[int32(mid)] = struct{}{}
				}
			}
		}
		if len(seeds) > 0 {
			bornMgr.Store(cores_declarations.BornBlockID(i), seeds)
		}
	}
	mapDataManager.BornBlockManager = bornMgr
}

func (mm *MapManager) Stop() {}

// Start start
func (mm *MapManager) Start() {
	go mm.loopTickCheck()
	go mm.loopTickAccept()
}

func (mm *MapManager) loopTickCheck() {
	ticker := time.NewTicker(time.Millisecond * 100) //
	defer ticker.Stop()
	mapClearTicker := time.NewTicker(time.Millisecond * 300)
	defer mapClearTicker.Stop()
	secondTicker := time.NewTicker(time.Second)
	defer secondTicker.Stop()

	for {
		select {
		case <-mm.opts.stopChan:
			return
		case <-ticker.C: // 行军清理
			nowTime := time.Now().Unix()

			// 地图推送处理
			go mm.upMapAsync()

			// 行军处理
			mm.timeMarchLock.Lock()
			marchIDs, ok := mm.timeMarch[nowTime]
			delete(mm.timeMarch, nowTime)
			mm.timeMarchLock.Unlock()
			if ok {
				for marchID := range marchIDs {
					go mm.marchDoFunc(mm, marchID)
				}
			}
		case <-mapClearTicker.C: // 地图清理
			nowTime := time.Now().Unix()

			// 清理地图
			mm.timeMapLock.Lock()
			mapIDs, ok := mm.timeMap[nowTime]
			delete(mm.timeMap, nowTime)
			mm.timeMapLock.Unlock()
			if ok {
				if len(mapIDs) > 0 {
					go mm.clearMapFunc(mapIDs, nowTime)
				}
			}
		case <-secondTicker.C: // 兜底清理
			nowTime := time.Now().Unix()

			// 行军处理(兜底处理)
			mm.timeMarchLock.Lock()
			marchIDs := make(map[cores_declarations.MarchID]struct{})
			for runTime, list := range mm.timeMarch {
				if runTime <= nowTime {
					for marchID := range list {
						marchIDs[marchID] = struct{}{}
					}
					delete(mm.timeMarch, runTime)
				}
			}
			mm.timeMarchLock.Unlock()
			for marchID := range marchIDs {
				go mm.marchDoFunc(mm, marchID)
			}

			// 清理地图
			mm.timeMapLock.Lock()
			mapIDs := make(map[cores_declarations.MapID]struct{})
			for runTime, list := range mm.timeMap {
				if runTime <= nowTime {
					for mapID := range list {
						mapIDs[mapID] = struct{}{}
					}
					delete(mm.timeMap, runTime)
				}
			}
			mm.timeMapLock.Unlock()
			if len(mapIDs) > 0 {
				go mm.clearMapFunc(mapIDs, nowTime)
			}
		}

	}
}

func (mm *MapManager) loopTickAccept() {
	for {
		select {
		case <-mm.opts.stopChan:
			return
		case marchInfo, ok := <-mm.GetMarchManage().TickerChan:
			if !ok {
				return
			}
			go mm.TickerAddMarch(marchInfo.GetMarchID(), marchInfo.GetEndTimeUx())
		}
	}
}

func (mm *MapManager) clearMapFunc(mapIDs map[cores_declarations.MapID]struct{}, nowTime int64) {

}
