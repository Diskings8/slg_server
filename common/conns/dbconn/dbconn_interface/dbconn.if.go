package dbconn_interface

import "server.slg.com/common/models/db_model_interface"

type DbcI interface {
	AutoMigrate(model db_model_interface.DbIModel) error
}
