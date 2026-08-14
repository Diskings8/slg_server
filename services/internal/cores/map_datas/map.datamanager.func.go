package map_datas

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_aois"
	"server.slg.com/services/internal/cores/roles"
)

// registerAsyncSaveOnce 保证 MapDataManager 的异步保存实体只注册一次
var registerAsyncSaveOnce sync.Once

// roleBriefSaveFunc 放置主城后回写角色简略数据。
// 包级函数变量，便于测试注入替换（默认实现访问角色 poller，依赖 Redis/DB）。
var roleBriefSaveFunc = func(brief *pb_role.RoleBrief) {
	if roles.GetPollerMgr() == nil {
		// 角色数据 poller 未初始化（单测环境），跳过回写
		return
	}
	roleData, releaseFunc, saveFunc, err := roles.Get(brief.GetRoleBaseInfo().GetSimpleInfo().GetRoleId())
	if err != nil {
		loggers.Logger.Error(err.Error())
		return
	}
	roleData.GetBrief().RoleBrief = brief
	saveFunc()
	releaseFunc()
}

// NewMapDataManager 创建并初始化地图数据管理器（稠密全量地图）
//   - mapConfig: 地图配置
//   - tableName: 数据库表名
//
// 自动创建 AOI、初始化全量格子坐标，调用 Init 注册到 AOI
func NewMapDataManager(mapConfig cores_declarations.MapConfigI, tableName string) *MapDataManager {
	aoi := map_aois.NewAoi(mapConfig)

	mdm := &MapDataManager{
		config:    mapConfig,
		tableName: tableName,
		AOI:       aoi,
	}

	count := mapConfig.MapCount()
	bigMap := make([]MapInfo, count)
	for id := range count {
		x, y := mapConfig.MapID2XY(cores_declarations.MapID(id))
		info := &bigMap[id]
		info.mapID = cores_declarations.MapID(id)
		info.coreMapID = cores_declarations.MapID(id)
		info.x = int(x)
		info.y = int(y)
	}

	mdm.Init(bigMap)
	return mdm
}
func (mdm *MapDataManager) Clear(mapIDs []cores_declarations.MapID, isNeedLock bool) {
	mdm.ClearMapInfoSlice(mdm.GetMapInfoSlice(mapIDs), isNeedLock)
}

// ClearMapInfoSlice 清空地块
func (mdm *MapDataManager) ClearMapInfoSlice(dataSlice []*MapInfo, lock bool) {
	if lock {
		for _, v := range dataSlice {
			v.Lock()
		}
		defer mdm.UnLock(dataSlice)
	}

	clearTime := time.Now()
	for _, d := range dataSlice {
		mdm.Free(d, false, clearTime)
	}
}

// Free 清空地块
func (mdm *MapDataManager) Free(mapInfo *MapInfo, lock bool, clearTime time.Time) {
	if lock {
		mapInfo.Lock()
		defer mapInfo.UnLock()
	}
	mapInfo.Free(clearTime)
	mdm.Save(mapInfo)
}

func (mdm *MapDataManager) SetRoleMainCity(roleCityState cores_declarations.RoleMainCityState, dataSlice []*MapInfo, roleBrief *pb_role.RoleBrief) error {
	var coreIndex int
	switch roleCityState {
	case cores_declarations.RoleMainCityStateNormal:
		if len(dataSlice) != cores_declarations.RoleMainCityStateNormalCoverCount {
			return errors.New("地块数量不对")
		}
		// 3×3 主城核心在中心（index 4），须与 GetFreeBorn 的 coreMapID 一致
		coreIndex = cores_declarations.Land3CoverBaseKey
	default:
		if len(dataSlice) != cores_declarations.RoleMainCityStatePortableCoverCount {
			return errors.New("地块数量不对")
		}
		// 单格便携主城，唯一元素即核心
		coreIndex = 0
	}
	// 检测位置可使用情况
	if !CheckRoleBornSiteSafeByMapInfos(false, dataSlice...) {
		return errors.New("地块校验不合法")
	}

	//
	updateNow := time.Now()
	coreMapInfo := dataSlice[coreIndex]
	for _, mapInfo := range dataSlice {
		// dataSlice 已由调用方持有写锁（GetFreeBorn 分配），用 freeLocked 避免重入死锁
		mapInfo.freeLocked(updateNow)
		mapInfo.serverID = roleBrief.GetRoleBaseInfo().GetSimpleInfo().GetServerId()
		mapInfo.ownerID = roleBrief.GetRoleBaseInfo().GetSimpleInfo().GetRoleId()
		mapInfo.coreMapID = coreMapInfo.mapID

		// aoi 更新
	}
	mdm.Save(dataSlice...)

	// 更新角色数据（函数变量，测试可注入替换）
	roleBriefSaveFunc(roleBrief)
	// aoi更新
	return nil
}

// GetFreeBorn 可用出生点,失败要调用freeBornFunc放回到出生块里
// 注意：取出的mapSlice是带锁的
func (mdm *MapDataManager) GetFreeBorn() (mapIDs []cores_declarations.MapID, lockMapSlice LockMapSlice, bornID cores_declarations.BornBlockID, coreMapID cores_declarations.MapID, freeBornFunc func(), err error) {
	mdm.BornBlockManager.Range(func(bornIDTmp cores_declarations.BornBlockID, v map[int32]struct{}) bool {
		// 随机找一个四块地都是空地
		for mapID := range v {
			mapIDsTmp := mdm.GetConfig().CoverMapIDs(mapID, 1, cores_declarations.HallLandCover/2)
			if len(mapIDsTmp) != cores_declarations.HallCoverCount {
				continue
			}

			// 尝试上锁当前种子附件的地块
			mapSliceTmp := mdm.GetMapInfoSlice(mapIDsTmp)
			if !mdm.TryLock(mapSliceTmp) {
				continue
			}
			// 判断已上锁的地块数是否和所需一致
			if len(mapSliceTmp) != len(mapIDsTmp) {
				mdm.UnLock(mapSliceTmp)
				return true
			}

			if CheckRoleBornSiteSafeByMapInfos(false, mapSliceTmp...) {
				// 赋值返回的数据
				mapIDs = mapIDsTmp
				bornID = bornIDTmp
				lockMapSlice = LockMapSlice{
					data: mapSliceTmp,
					mdm:  mdm,
				}
				coreMapID = mapIDsTmp[cores_declarations.Land3CoverBaseKey]

				// 不再整块 Use：出生只占 3×3（已 TryLock 独占），同一块内不同种子可容纳多个玩家。
				// 块保持空闲池中，随城市增多 3×3 被占满的种子自然不可用，容量由 3×3 锁保证。
				freeBornFunc = func() {}

				return false
			}
			mdm.UnLock(mapSliceTmp)
		}

		if common_globals.IsDev() {
			// 这个地块一个9块空地都找不到，直接将这块地置为不可创建，打印出错误提示
			loggers.Logger.Warn("在一个可放置玩家的位置，找不到一个1格相连的空地，打印出错误信息", zap.Int32("coreMapID", int32(coreMapID)), zap.Any("v", v))
		}

		mdm.BornBlockManager.Delete(bornIDTmp)

		return true
	})

	if bornID < 1 {
		err = errors.New("没有空佘的位置可创号")
	}
	return mapIDs, lockMapSlice, bornID, coreMapID, freeBornFunc, err
}

// Around 取得周围的地块数据,不包含中心地块
func (mdm *MapDataManager) Around(mapID cores_declarations.MapID) (resp []*MapInfo) {
	resp = mdm.filterAround(mapID, [][2]int32{{-1, 0}, {-1, 1}, {-1, -1}, {0, -1}, {0, 1}, {1, -1}, {1, 0}, {1, 1}})
	return
}

// AroundCover 取得周围的地块数据,包含本地块，共9个
func (mdm *MapDataManager) AroundCover(mapID cores_declarations.MapID) (resp []*MapInfo) {
	resp = mdm.filterAround(mapID, [][2]int32{{-1, 0}, {-1, 1}, {-1, -1}, {0, -1}, {0, 0}, {0, 1}, {1, -1}, {1, 0}, {1, 1}})
	return
}

func (mdm *MapDataManager) filterAround(mapID cores_declarations.MapID, filter [][2]int32) (resp []*MapInfo) {
	x, y := mdm.config.MapID2XY(mapID)
	for _, v := range filter {
		loopMapId := mdm.config.XY2MapID(x+v[0], y+v[1])
		if tmp, ok := mdm.GetMapInfo(loopMapId); ok {
			resp = append(resp, tmp)
		}
	}
	return
}
