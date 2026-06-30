package dbconn

import "server.slg.com/common/conns/dbconn/interface_dbconn"

func InitDB(dbType string, dsn string) interface_dbconn.DbcI{
	switch dbType {
	case DbType_Mysql:
		mysqlD,err :=
	}
}
