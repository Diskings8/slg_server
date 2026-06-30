package interface_dbconn

import "server.slg.com/common/models/db_interface_model"

type DbcI interface {
	AutoMigrate(model db_interface_model.DbIModel) error
}
