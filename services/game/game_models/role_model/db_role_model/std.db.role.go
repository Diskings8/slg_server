package db_role_model

import (
	"server.slg.com/common/common_declarations"
)

var _ common_declarations.DbModelI = (*RoleDb)(nil)

// RoleDb 角色数据库模型，映射数据库中的角色表结构
type RoleDb struct {
	Id uint64 `gorm:"primary_key;column:id;type:bigint(20);not null"`
}

func (r RoleDb) TableName() string {
	return "Role"
}
