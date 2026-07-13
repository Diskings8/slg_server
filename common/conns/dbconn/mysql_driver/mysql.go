package mysql_driver

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
)

var _ common_declarations.DbcI = (*MysqlDriver)(nil)

// MysqlDriver MySQL 数据库驱动实现，基于 GORM 封装，提供连接池配置和自动迁移能力
type MysqlDriver struct {
	db *gorm.DB
}

func (m MysqlDriver) Table(tableName string) common_declarations.DbcI {
	//TODO implement me
	panic("implement me")
}

func (m MysqlDriver) Find(model common_declarations.DbModelI) error {
	//TODO implement me
	panic("implement me")
}

func NewDriver(dsn string) (*MysqlDriver, error) {
	orm, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := orm.DB()
	if err != nil {
		return nil, err
	}

	// 配置连接池（根据你的服务器配置调整）
	sqlDB.SetMaxOpenConns(100)                 // 最大同时打开的连接数
	sqlDB.SetMaxIdleConns(50)                  // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(30 * time.Second) // 连接最长存活时间
	sqlDB.SetConnMaxIdleTime(10 * time.Second) // 空闲连接最长存活时间

	driver := &MysqlDriver{db: orm}
	return driver, nil
}

func (m MysqlDriver) AutoMigrate(model common_declarations.DbModelI) error {
	return m.db.AutoMigrate(model)
}
