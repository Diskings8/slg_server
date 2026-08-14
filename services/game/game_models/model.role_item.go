package game_models

import (
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/models"
)

const role_item = "role_item"

// RoleItem 角色道具（背包）
type RoleItem struct {
	models.ModelBase
	RoleID      uint64               `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_item,priority:1;comment:角色ID"`
	ConfigID    pb_confs.ItemID      `gorm:"column:config_id;type:int(11);not null;uniqueIndex:idx_role_item,priority:2;comment:道具配置ID"`
	ItemType    pb_confs.ItemType    `gorm:"column:item_type;type:int(11);not null;default:0;index;comment:道具类型"`
	ItemSubType pb_confs.ItemSubType `gorm:"column:item_sub_type;type:int(11);not null;default:0;index;comment:道具子类型"`
	Count       int64                `gorm:"column:count;type:bigint(20);not null;default:0;comment:数量"`
}

func (RoleItem) TableName() string {
	return role_item
}

func NewRoleItem(roleID uint64, item common_declarations.ItemUse, ID uint64, creatUx int64) *RoleItem {
	return &RoleItem{
		ModelBase:   models.ModelBase{ID: ID, CreatedAt: creatUx},
		RoleID:      roleID,
		ConfigID:    item.ItemID,
		ItemType:    item.ItemType,
		ItemSubType: item.ItemSubType,
		Count:       item.Count,
	}
}
