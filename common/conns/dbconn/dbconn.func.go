package dbconn

import (
	"errors"
	"fmt"

	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn/mysql_driver"
)

// InitDB 初始化数据库连接，分别设置读写库
//
// writeDsn 和 readDsn 可以相同（单库模式），也可以不同（读写分离）。
// 后续通过 GetWriteDbConn() / GetReadDbConn() 获取连接。
func InitDB(dbType, writeDsn, readDsn string) error {
	if writeDsn == "" {
		return errors.New("write dsn is required")
	}

	switch dbType {
	case common_declarations.DbTypeMysql:
		w, err := mysql_driver.NewDriver(writeDsn)
		if err != nil {
			return fmt.Errorf("init write db: %w", err)
		}
		writeDb = w

		if readDsn != "" && readDsn != writeDsn {
			r, err := mysql_driver.NewDriver(readDsn)
			if err != nil {
				return fmt.Errorf("init read db: %w", err)
			}
			readDb = r
		} else {
			readDb = w // 未指定读库则复用写库
		}
		return nil
	default:
		return fmt.Errorf("unsupported db type: %s", dbType)
	}
}

// MustInitDB InitDB 的 panic 版本，用于项目启动时直接初始化
func MustInitDB(dbType, writeDsn, readDsn string) {
	if err := InitDB(dbType, writeDsn, readDsn); err != nil {
		panic(fmt.Sprintf("init db failed: %v", err))
	}
}
