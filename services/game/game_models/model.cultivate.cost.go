package game_models

import (
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/common/models"
)

const cultivate_cost = "cultivate_cost"

// CultivateCost 养成消耗
type CultivateCost struct {
	models.ModelBase
	RoleID        uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_cultivate_cost"`
	Cost          []*pb_common.Int32KV     `gorm:"serializer:json;type:json;not null"`
	CultivateType pb_cultivate.CultivateType `gorm:"column:cultivate_type;type:int(11);not null;default:0"`
}

func (CultivateCost) TableName() string {
	return cultivate_cost
}
