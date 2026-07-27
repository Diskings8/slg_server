package game_roles

import "server.slg.com/common/common_declarations"

var _ common_declarations.DbModelI = (*RoleDbModel)(nil)

// RoleDbModel 角色数据库模型，映射数据库中的角色表结构
type RoleDbModel struct {
	ID uint64 `gorm:"column:id;type:bigint(20) unsigned;primaryKey;autoIncrement:false"`
}

func (r *RoleDbModel) TableName() string {
	return "role"
}
