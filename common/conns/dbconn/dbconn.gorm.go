package dbconn

import (
	"gorm.io/gorm"
	"server.slg.com/common/conns/dbconn/mysql_driver"
)

// GormDB 返回写库底层 GORM 实例。
//
// DbcI 是精简查询接口，缺少分页/排序/计数（Order/Limit/Offset/Count）。
// 需要完整 GORM 能力的模块（如战斗记录分页查询）可直接用 GormDB()。
// 返回 nil 表示写库未初始化或非 MySQL 驱动。
func GormDB() *gorm.DB {
	if d, ok := writeDb.(*mysql_driver.MysqlDriver); ok {
		return d.DB()
	}
	return nil
}
