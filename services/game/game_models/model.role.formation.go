package game_models

import (
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/models"
)

const role_formation = "role_formation"

// RoleFormation 已上阵队伍（队列，归属城市）
// 唯一 ID 即队列ID（snowflake 分配），校场等级决定队列数
// 英雄槽位复用 maps.march.HeroSlot（proto 定义，前端共用）
type RoleFormation struct {
	models.ModelBase
	RoleID    uint64                    `gorm:"column:role_id;type:bigint(20);not null;index;comment:角色ID"` // 角色ID
	CityID    uint64                    `gorm:"column:city_id;type:bigint(20);not null;index;comment:归属城市ID"` // 归属城市
	HeroSlots []*pb_maps_march.HeroSlot `gorm:"serializer:json;type:json;not null;comment:英雄槽位(数量来自配置,默认3)"` // 英雄槽位（数量来自配置，默认3）
}

func (RoleFormation) TableName() string {
	return role_formation
}
