package db_role_model

import (
	"server.slg.com/common/models/db_interface_model"
)

var _ db_interface_model.DbIModel = (*RoleDb)(nil)

type RoleDb struct {
	Id uint64 `gorm:"primary_key;column:id;type:bigint(20);not null"`
}

func (r RoleDb) TableName() string {
	return "Role"
}
