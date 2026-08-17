package role_resource_tiles

import (
	"go.uber.org/zap"
	"server.slg.com/common/loggers"
	"server.slg.com/common/models"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

// NewRoleResourceTiles 创建资源地产出快照集合
func NewRoleResourceTiles(roleID uint64) *RoleResourceTiles {
	return &RoleResourceTiles{
		RoleID: roleID,
		List:   make([]*game_models.RoleResourceTile, 0),
	}
}

// Init 初始化 Mem 索引（按地块 MapID）
func (rts *RoleResourceTiles) Init() {
	for _, modelOne := range rts.List {
		rts.Mem.Store(modelOne.MapID, NewRoleResourceTile(modelOne))
	}
}

// Copy 深拷贝（副本模式）
func (rts *RoleResourceTiles) Copy(src *RoleResourceTiles) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}
	err = util_jsons.Unmarshal(b, rts)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}
	rts.Init()
}

// NewRoleResourceTile 包装单个快照
func NewRoleResourceTile(one *game_models.RoleResourceTile) *RoleResourceTile {
	return &RoleResourceTile{RoleResourceTile: one}
}

// Get 按地块 MapID 查询快照
func (rts *RoleResourceTiles) Get(mapID int32) *RoleResourceTile {
	if v, ok := rts.Mem.Load(mapID); ok {
		return v
	}
	return nil
}

// Upsert 新增或更新地块快照；返回是否有变化。
//
// 新增（首占/攻占）：记 LastSettleUx=now，产出从此刻起算；
// 更新（开发升级）：由调用方先 SettleRoleResources 结算旧等级，再传 now 重置结算起点。
func (rts *RoleResourceTiles) Upsert(roleID uint64, mapID int32, level, elementType int32, nowUx int64) bool {
	if exist, ok := rts.Mem.Load(mapID); ok {
		if exist.Level == level && exist.ElementType == elementType {
			return false
		}
		exist.Level = level
		exist.ElementType = elementType
		exist.LastSettleUx = nowUx
		return true
	}
	modelOne := &game_models.RoleResourceTile{
		ModelBase: models.ModelBase{
			ID:        snowflakes.GenUUID(),
			CreatedAt: nowUx,
			UpdatedAt: nowUx,
		},
		RoleID:       roleID,
		MapID:        mapID,
		Level:        level,
		ElementType:  elementType,
		LastSettleUx: nowUx,
	}
	rts.List = append(rts.List, modelOne)
	rts.Mem.Store(mapID, NewRoleResourceTile(modelOne))
	return true
}

// Remove 移除地块快照（放弃/被夺/非资源）；返回是否有变化
func (rts *RoleResourceTiles) Remove(mapID int32) bool {
	if _, ok := rts.Mem.Load(mapID); !ok {
		return false
	}
	rts.Mem.Delete(mapID)
	for i, v := range rts.List {
		if v.MapID == mapID {
			rts.List = append(rts.List[:i], rts.List[i+1:]...)
			break
		}
	}
	return true
}
