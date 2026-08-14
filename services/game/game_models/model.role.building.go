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
	RoleID    uint64                    `gorm:"column:role_id;type:bigint(20);not null;index:idx_role_building;comment:角色ID"`
	Type      pb_city.BuildingType      `gorm:"column:type;type:int(11);not null;comment:建筑类型(city.proto枚举)"`
	Footprint pb_city.BuildingFootprint `gorm:"column:footprint;type:int(11);not null;comment:占地(city.proto枚举)"`
	MapID     int32                     `gorm:"column:map_id;type:int(11);not null;comment:中心格"` // 中心格
	Level     uint32                    `gorm:"column:level;type:int(11) unsigned;not null;default:0;comment:当前生效等级(建造中=0)"` // 当前生效等级（建造中=0）
	State     pb_city.BuildingState     `gorm:"column:state;type:int(11);not null;default:0;comment:建筑状态"` // 建筑状态
	CityID    uint64                    `gorm:"column:city_id;type:bigint(20);not null;default:0;comment:归属城市(主城为0)"` // 归属城市（校场/兵营等附属建筑；主城为 0）
	NextLevel uint32                    `gorm:"column:next_level;type:int(11) unsigned;not null;default:0;comment:建造/升级目标等级(Completed为0)"` // 建造/升级目标等级（Completed 为 0）
	EndTimeUx int64                     `gorm:"column:end_time_ux;type:bigint(20);not null;default:0;comment:完成时间戳(Completed为0)"` // 完成时间戳（Completed 为 0）
}

func (RoleBuilding) TableName() string {
	return role_building
}
