package game_models

import (
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/common/models"
)

const role_building = "role_building"

// RoleBuilding 角色建筑（主城/分城/军事建筑统一）
// Type/Footprint 枚举定义在 city.proto，前端共用
type RoleBuilding struct {
	models.ModelBase
	RoleID    uint64                    `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_building"`
	Type      pb_city.BuildingType      `gorm:"column:type;type:int(11);not null"`
	Footprint pb_city.BuildingFootprint `gorm:"column:footprint;type:int(11);not null"`
	MapID     int32                     `gorm:"column:map_id;type:int(11);not null"` // 中心格
	Level     uint32                    `gorm:"column:level;type:int(11) unsigned;not null;default:1"`
	State     pb_city.BuildingState     `gorm:"column:state;type:int(11);not null;default:0"` // 建筑状态
	CityID    uint64                    `gorm:"column:city_id;type:bigint(20);not null;default:0"` // 归属城市（兵营等附属建筑；主城/分城为 0）
}

func (RoleBuilding) TableName() string {
	return role_building
}
