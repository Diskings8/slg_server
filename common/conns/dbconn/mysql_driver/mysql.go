package mysql_driver

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
)

var _ common_declarations.DbcI = (*MysqlDriver)(nil)

// MysqlDriver MySQL 数据库驱动，基于 GORM 封装，支持链式调用
//
// 调用方式：
//
//	dbconn.GetWriteDbConn().Table("table_name").Create(data).Error()
//	dbconn.GetReadDbConn().Table("table_name").Where("id = ?", 1).Take(&result).Error()
type MysqlDriver struct {
	db    *gorm.DB
	table string
	err   error
}

func NewDriver(dsn string) (*MysqlDriver, error) {
	orm, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := orm.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetMaxIdleConns(50)
	sqlDB.SetConnMaxLifetime(30 * time.Second)
	sqlDB.SetConnMaxIdleTime(10 * time.Second)

	return &MysqlDriver{db: orm}, nil
}

// ---- 链式调用方法 ----

func (m *MysqlDriver) Table(tableName string) common_declarations.DbcI {
	m.table = tableName
	return m
}

func (m *MysqlDriver) Where(query any, args ...any) common_declarations.DbcI {
	m.db = m.db.Where(query, args...)
	return m
}

func (m *MysqlDriver) Find(model any) common_declarations.DbcI {
	m.err = m.session().Find(model).Error
	return m
}

func (m *MysqlDriver) Take(model any, query ...any) common_declarations.DbcI {
	m.err = m.session().Take(model, query...).Error
	return m
}

func (m *MysqlDriver) Create(model any) common_declarations.DbcI {
	m.err = m.session().Create(model).Error
	return m
}

func (m *MysqlDriver) CreateInBatches(model any, batchSize int) common_declarations.DbcI {
	m.err = m.session().CreateInBatches(model, batchSize).Error
	return m
}

func (m *MysqlDriver) Save(model any) common_declarations.DbcI {
	m.err = m.session().Save(model).Error
	return m
}

func (m *MysqlDriver) Delete(model any, query ...any) common_declarations.DbcI {
	m.err = m.session().Delete(model, query...).Error
	return m
}

// ---- 非链式方法 ----

func (m *MysqlDriver) Error() error {
	return m.err
}

func (m *MysqlDriver) AutoMigrate(model common_declarations.DbModelI) error {
	return m.db.AutoMigrate(model)
}

// Transaction 在事务中执行 fn
//
//	fn 返回 nil → 提交；返回 error → 回滚。
//	fn 中通过 tx 执行的查询全部在同一个事务内。
//
//	用法：
//		dbconn.GetWriteDbConn().Transaction(func(tx dbconn.DbcI) error {
//			tx.Table("a").Create(&data1)
//			tx.Table("b").Create(&data2)
//			return nil
//		})
func (m *MysqlDriver) Transaction(fn func(tx common_declarations.DbcI) error) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		txDriver := &MysqlDriver{db: tx}
		return fn(txDriver)
	})
}

// session 创建基于当前 table 的 GORM session，不影响原始 db
func (m *MysqlDriver) session() *gorm.DB {
	sess := m.db.Session(&gorm.Session{})
	if m.table != "" {
		sess = sess.Table(m.table)
	}
	return sess
}
