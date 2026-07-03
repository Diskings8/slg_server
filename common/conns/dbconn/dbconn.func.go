package dbconn

import (
	"errors"

	"server.slg.com/common/conns/dbconn/dbconn_interface"
	"server.slg.com/common/conns/dbconn/mysql_driver"
)

func InitDB(dbType string, dsn string) (dbconn_interface.DbcI, error) {
	switch dbType {
	case DbType_Mysql:
		return mysql_driver.NewDriver(dsn)
	}
	return nil, errors.New("not set db type")
}
