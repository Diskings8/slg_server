package dbconn

import (
	"server.slg.com/common/common_declarations"
)

const (
	MaxSaveLen = 1000
)

var readDb common_declarations.DbcI
var writeDb common_declarations.DbcI

func GetWriteDbConn() common_declarations.DbcI {
	return writeDb
}

func GetReadDbConn() common_declarations.DbcI {
	return readDb
}
