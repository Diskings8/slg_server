package dbconn

import "server.slg.com/common/conns/dbconn/dbconn_interface"

func InitDB(dbType string, dsn string) dbconn_interface.DbcI{
	switch dbType {
	case DbType_Mysql:
		mysqlD,err :=
	}
}
