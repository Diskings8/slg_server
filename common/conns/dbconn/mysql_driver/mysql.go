package mysql_driver

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"server.slg.com/common/conns/dbconn/dbconn_interface"
	"server.slg.com/common/models/db_model_interface"
)

var _ dbconn_interface.DbcI = (*MysqlDriver)(nil)

type MysqlDriver struct {
	db *gorm.DB
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

func (m MysqlDriver) AutoMigrate(model db_model_interface.DbIModel) error {
	return m.db.AutoMigrate(model)
}
