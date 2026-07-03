package dbconn_interface

type DbcI interface {
	AutoMigrate(model any) error
	Table(tableName string) DbcI
	Find(model any) error
}
