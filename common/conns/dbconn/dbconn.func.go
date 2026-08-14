package dbconn

import (
	"database/sql"
	"errors"
	"fmt"

	gomysql "github.com/go-sql-driver/mysql"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn/mysql_driver"
)

// EnsureDatabase 确保 DSN 指向的数据库存在，不存在则自动创建。
//
// 连接 MySQL 服务端（不带库名）执行 CREATE DATABASE IF NOT EXISTS，
// 使各服务启动时自建库，不再依赖外部 init.sql / 手动建库脚本。
func EnsureDatabase(dsn string) error {
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("parse dsn: %w", err)
	}
	if cfg.DBName == "" {
		return nil
	}

	// 克隆配置并去掉库名，连接到服务端（不选默认 schema）
	serverCfg := *cfg
	serverCfg.DBName = ""

	db, err := sql.Open("mysql", serverCfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("open server conn: %w", err)
	}
	defer db.Close()

	createSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
		cfg.DBName,
	)
	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("create database %s: %w", cfg.DBName, err)
	}
	return nil
}

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
		// 服务自建库：目标库不存在时先 CREATE DATABASE IF NOT EXISTS
		if err := EnsureDatabase(writeDsn); err != nil {
			return fmt.Errorf("ensure write db: %w", err)
		}
		if readDsn != "" && readDsn != writeDsn {
			if err := EnsureDatabase(readDsn); err != nil {
				return fmt.Errorf("ensure read db: %w", err)
			}
		}

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
