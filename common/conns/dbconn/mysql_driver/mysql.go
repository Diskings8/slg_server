package mysql_driver

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
)

var _ common_declarations.DbcI = (*MysqlDriver)(nil)

// MysqlDriver MySQL 数据库驱动，实现 DbcI 接口。
//
// 无状态：链式方法累积到 stmt，终态方法在 base 的新 session 上执行并 reset。
// 因此同一个实例（如 dbconn.GetWriteDbConn() 单例）可被多个 store 安全地多次链式调用，不会串条件。
type MysqlDriver struct {
	base  *gorm.DB // 连接（永不修改）
	stmt  *gorm.DB // 当前操作累积的链（nil = 从 base 起新链）
	table string   // 当前操作的 Table 覆盖
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

	return &MysqlDriver{base: orm}, nil
}

// DB 暴露底层 GORM 实例，供需要完整 GORM 能力（分页/排序/计数）的调用方使用
func (m *MysqlDriver) DB() *gorm.DB { return m.base }

// cur 返回当前链（无则从 base 起新 session，并应用 Table 覆盖）
func (m *MysqlDriver) cur() *gorm.DB {
	if m.stmt != nil {
		return m.stmt
	}
	s := m.base.Session(&gorm.Session{NewDB: true})
	if m.table != "" {
		s = s.Table(m.table)
	}
	return s
}

// reset 终态方法执行后清空当前操作状态
func (m *MysqlDriver) reset() {
	m.stmt = nil
	m.table = ""
}

// ---- 链式方法 ----

func (m *MysqlDriver) Table(tableName string) common_declarations.DbcI {
	m.table = tableName
	return m
}

func (m *MysqlDriver) Where(query any, args ...any) common_declarations.DbcI {
	m.stmt = m.cur().Where(query, args...)
	return m
}

func (m *MysqlDriver) Model(model any) common_declarations.DbcI {
	m.stmt = m.cur().Model(model)
	return m
}

func (m *MysqlDriver) Order(cond string) common_declarations.DbcI {
	m.stmt = m.cur().Order(cond)
	return m
}

func (m *MysqlDriver) Limit(limit int) common_declarations.DbcI {
	m.stmt = m.cur().Limit(limit)
	return m
}

func (m *MysqlDriver) Offset(offset int) common_declarations.DbcI {
	m.stmt = m.cur().Offset(offset)
	return m
}

// ---- 终态方法（执行当前链后 reset，错误经 Error() 取） ----

func (m *MysqlDriver) Find(model any) common_declarations.DbcI {
	m.err = m.cur().Find(model).Error
	m.reset()
	return m
}

func (m *MysqlDriver) First(model any, query ...any) common_declarations.DbcI {
	m.err = m.cur().First(model, query...).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Take(model any, query ...any) common_declarations.DbcI {
	m.err = m.cur().Take(model, query...).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Create(model any) common_declarations.DbcI {
	m.err = m.cur().Create(model).Error
	m.reset()
	return m
}

func (m *MysqlDriver) CreateInBatches(model any, batchSize int) common_declarations.DbcI {
	m.err = m.cur().CreateInBatches(model, batchSize).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Save(model any) common_declarations.DbcI {
	m.err = m.cur().Save(model).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Delete(model any, query ...any) common_declarations.DbcI {
	m.err = m.cur().Delete(model, query...).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Updates(values any) common_declarations.DbcI {
	m.err = m.cur().Updates(values).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Update(column string, value any) common_declarations.DbcI {
	m.err = m.cur().Update(column, value).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Count(count *int64) common_declarations.DbcI {
	m.err = m.cur().Count(count).Error
	m.reset()
	return m
}

func (m *MysqlDriver) Exec(sql string, values ...any) common_declarations.DbcI {
	m.err = m.base.Exec(sql, values...).Error
	m.reset()
	return m
}

// ---- 非链式方法 ----

func (m *MysqlDriver) Error() error { return m.err }

// AutoMigrate 变参建表（幂等）
func (m *MysqlDriver) AutoMigrate(models ...common_declarations.DbModelI) error {
	anyModels := make([]any, len(models))
	for i, mm := range models {
		anyModels[i] = mm
	}
	return m.base.AutoMigrate(anyModels...)
}

// Transaction 在事务中执行 fn（fn 内通过 tx 的查询在同一事务内）
func (m *MysqlDriver) Transaction(fn func(tx common_declarations.DbcI) error) error {
	return m.base.Transaction(func(tx *gorm.DB) error {
		return fn(&MysqlDriver{base: tx})
	})
}
