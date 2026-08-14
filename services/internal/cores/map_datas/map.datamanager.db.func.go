package map_datas

import (
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"

	"go.uber.org/zap"
)

// MapInfoDB map_data 表持久化 DTO。
//
// 说明：MapInfo 的持久化字段几乎全部未导出且无 gorm tag，直接 Save 无法正确落库，
// 因此用独立 DTO 承载需要持久化的动态状态（元素/归属/等级/保护期等）。
// 锁与指针字段（overlayBuilding/overlayEvent）本轮不持久化，待建筑系统接入后再补。
type MapInfoDB struct {
	MapID            int32  `gorm:"column:map_id;primaryKey;comment:地图格子ID"`
	ElementType      int32  `gorm:"column:element_type;comment:元素类型(地形/资源)"`
	ConfigID         uint32 `gorm:"column:config_id;comment:元素配置ID"`
	Level            int32  `gorm:"column:level;comment:元素等级"`
	ServerID         uint32 `gorm:"column:server_id;comment:所属服务器ID"`
	OwnerID          uint64 `gorm:"column:owner_id;comment:归属角色ID(0=无主)"`
	IsDeveloped      bool   `gorm:"column:is_developed;comment:是否已开发"`
	ProtectedEndTime int64  `gorm:"column:protected_end_time;comment:保护期结束时间"`
	CoreMapID        int32  `gorm:"column:core_map_id;comment:所属主城核心格ID"`
}

func (MapInfoDB) TableName() string { return "map_data" }

// toDB 将内存格子导出为持久化 DTO。
//
// 注意：调用方须已持有 mi 写锁（save 由 SaveDo 在 TryLock 后调用），
// 内部不再加锁 —— 同一 goroutine 持写锁再 RLock 会造成 RWMutex 重入死锁。
func (mi *MapInfo) toDB() MapInfoDB {
	return MapInfoDB{
		MapID:            int32(mi.mapID),
		ElementType:      int32(mi.ElementType),
		ConfigID:         mi.configID,
		Level:            int32(mi.Level),
		ServerID:         mi.serverID,
		OwnerID:          mi.ownerID,
		IsDeveloped:      mi.isDeveloped,
		ProtectedEndTime: mi.protectedEndTime,
		CoreMapID:        int32(mi.coreMapID),
	}
}

// Load 启动时从 DB 加载地图动态状态（稀疏覆盖模型）。
//
// 底图由种子确定性生成（worldmap_inits.InitMapElements），map_data 只存
// 「被修改过」的格子（Save 触发点：Free / SetRoleMainCity / Occupy / 开发升级）。
// 启动流程：先生成底图 → Load 用 DB 行覆盖 → 得到完整可用地图。
func (mdm *MapDataManager) Load(dbc common_declarations.DbcI) error {
	if err := dbc.Table(mdm.tableName).AutoMigrate(&MapInfoDB{}); err != nil {
		return err
	}

	var rows []MapInfoDB
	if err := dbc.Table(mdm.tableName).Find(&rows).Error(); err != nil {
		return err
	}

	for _, r := range rows {
		mi, ok := mdm.GetMapInfo(cores_declarations.MapID(r.MapID))
		if !ok {
			continue
		}
		mi.rwLock.Lock()
		mi.ElementType = cores_declarations.ElementType(r.ElementType)
		mi.configID = r.ConfigID
		mi.Level = cores_declarations.MapLevel(r.Level)
		mi.serverID = r.ServerID
		mi.ownerID = r.OwnerID
		mi.isDeveloped = r.IsDeveloped
		mi.protectedEndTime = r.ProtectedEndTime
		mi.coreMapID = cores_declarations.MapID(r.CoreMapID)
		mi.rwLock.Unlock()
	}

	loggers.Logger.Info("map data loaded", zap.Int("rows", len(rows)))
	return nil
}
