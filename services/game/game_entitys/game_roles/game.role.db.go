package game_roles

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/loggers"
)

var _ common_declarations.DBModuleI = new(Role)

func (r *Role) DBCreate(tx common_declarations.DbcI) error {
	writeDB := dbconn.GetWriteDbConn()
	if writeDB == nil {
		return fmt.Errorf("writeDB is nil, uuid: %d", r.ID)
	}

	return writeDB.Transaction(func(tx common_declarations.DbcI) error {
		val := reflect.ValueOf(r).Elem()
		for _, field := range val.Fields() {
			if !field.CanInterface() {
				continue
			}

			if module, ok := field.Interface().(common_declarations.DBModuleI); ok {
				err := module.DBCreate(tx)
				if err != nil {
					loggers.Logger.Error("DBCreate failed", zap.Uint64("uuid", r.ID), zap.Error(err))
					return err
				}
			}
		}

		return nil
	})
}

func (r *Role) DBDelete(tx common_declarations.DbcI) error {
	writeDB := dbconn.GetWriteDbConn()
	if writeDB == nil {
		return fmt.Errorf("writeDB is nil, uuid: %d", r.ID)
	}

	return writeDB.Transaction(func(tx common_declarations.DbcI) error {
		val := reflect.ValueOf(r).Elem()
		for _, field := range val.Fields() {
			if !field.CanInterface() {
				continue
			}

			if module, ok := field.Interface().(common_declarations.DBModuleI); ok {
				err := module.DBDelete(tx)
				if err != nil {
					loggers.Logger.Error("DBDelete failed", zap.Uint64("uuid", r.ID), zap.Error(err))
					return err
				}
			}
		}

		return nil
	})
}

func (r *Role) DBSave(tx common_declarations.DbcI) error {
	writeDB := dbconn.GetWriteDbConn()
	if writeDB == nil {
		return fmt.Errorf("writeDB is nil, uuid: %d", r.ID)
	}

	return writeDB.Transaction(func(tx common_declarations.DbcI) error {
		val := reflect.ValueOf(r).Elem()
		for _, field := range val.Fields() {
			if !field.CanInterface() {
				continue
			}

			if module, ok := field.Interface().(common_declarations.DBModuleI); ok {
				err := module.DBSave(tx)
				if err != nil {
					loggers.Logger.Error("DBSave failed", zap.Uint64("uuid", r.ID), zap.Error(err))
					return err
				}
			}
		}

		return nil
	})
}

func (r *Role) DBGet(tx common_declarations.DbcI) error {
	readDB := dbconn.GetReadDbConn()
	if readDB == nil {
		return fmt.Errorf("readDB is nil, uuid: %d", r.ID)
	}

	val := reflect.ValueOf(r).Elem()
	for _, field := range val.Fields() {
		if !field.CanInterface() {
			continue
		}

		if module, ok := field.Interface().(common_declarations.DBModuleI); ok {
			err := module.DBGet(readDB)
			if err != nil {
				// logger.Get().Error("DBGet failed", zap.Uint64("uuid", r.ID), zap.Error(err))
				return err
			}
		}
	}

	return nil
}
