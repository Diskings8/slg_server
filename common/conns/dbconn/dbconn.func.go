package dbconn

import (
	"errors"

	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn/mysql_driver"
)

func InitDB(nodeType, dbType string, dsn string) (common_declarations.DbcI, error) {
	switch dbType {
	case common_declarations.DbTypeMysql:
		return mysql_driver.NewDriver(dsn)
	}
	return nil, errors.New("not set db type")
}
