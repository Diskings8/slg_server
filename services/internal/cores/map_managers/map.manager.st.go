package map_managers

import (
	"sync"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_blocks"
	"server.slg.com/services/internal/cores/map_connects"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

// MapManager 地图管理器，管理地图数据、行军、时间事件和区块，是地图系统的核心调度单元
type MapManager struct {
	RoomID             uint64
	MapGroup           cores_declarations.MapGroup
	mapDataManager     *map_datas.MapDataManager
	roleConnectManager *map_connects.RoleConnectManager
	marchManage        *marchs.MarchInfoManager
	timeMarch          map[int64]map[cores_declarations.MarchID]struct{}
	timeMarchLock      sync.Mutex
	timeMap            map[int64]map[cores_declarations.MapID]struct{}
	timeMapLock        sync.Mutex
	marchDoFunc        func(*MapManager, cores_declarations.MarchID)
	marchDoFuncHandle  func(*MapManager, *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI
	mapBlock           *map_blocks.MapBlock
	opts               *options // 参数
	//
	waitUpdateMapID   map[cores_declarations.MapID]struct{}
	waitUpdateMapLock sync.Mutex

	battleSettleFunc cores_declarations.BattleSettleFunc // 战斗结算回调（worldmap 注入，内部调 battle 节点 RPC）
}

// SetBattleSettleFunc 注入战斗结算回调
func (mm *MapManager) SetBattleSettleFunc(f cores_declarations.BattleSettleFunc) {
	mm.battleSettleFunc = f
}

// GetBattleSettleFunc 获取战斗结算回调（未注入返回 nil，attack 到点将走召回兜底）
func (mm *MapManager) GetBattleSettleFunc() cores_declarations.BattleSettleFunc {
	return mm.battleSettleFunc
}

func (mm *MapManager) GetMapDataManager() *map_datas.MapDataManager {
	return mm.mapDataManager
}

func (mm *MapManager) GetRoleConnectManager() *map_connects.RoleConnectManager {
	return mm.roleConnectManager
}

func (mm *MapManager) GetMarchManage() *marchs.MarchInfoManager {
	return mm.marchManage
}

func (mm *MapManager) GetBlock() *map_blocks.MapBlock {
	return mm.mapBlock
}

func (mm *MapManager) GetConf() cores_declarations.MapConfigI {
	return mm.mapDataManager.GetConfig()
}
