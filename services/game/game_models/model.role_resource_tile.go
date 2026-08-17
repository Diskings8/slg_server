package game_models

import "server.slg.com/common/models"

const role_resource_tile = "role_resource_tile"

// RoleResourceTile 角色占领的资源地产出快照（惰性结算用）
//
// 世界事实在 worldmap（地块 owner/level/element），本表为 game 侧副本：
// worldmap 地块变更（开发升级/攻占/放弃）经 Redis Stream 事件同步到这里，
// 结算时按 elapsed × 每小时产量 → AddResource（cap 钳制），与建筑惰性结算同模式。
type RoleResourceTile struct {
	models.ModelBase
	RoleID       uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_tile,priority:1;comment:角色ID"`
	MapID        int32  `gorm:"column:map_id;type:int(11);not null;uniqueIndex:idx_role_tile,priority:2;comment:资源地块ID"`
	Level        int32  `gorm:"column:level;type:int(11);not null;default:0;comment:当前等级"`
	ElementType  int32  `gorm:"column:element_type;type:int(11);not null;default:0;comment:地块元素类型(Resources_1~4)"`
	LastSettleUx int64  `gorm:"column:last_settle_ux;type:bigint(20);not null;default:0;comment:上次结算时间戳"`
}

func (RoleResourceTile) TableName() string {
	return role_resource_tile
}
