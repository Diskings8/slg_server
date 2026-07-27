package game_roles

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
)

var _ common_declarations.DBModuleI = new(Role)

func (r *Role) DBCreate(writeDB common_declarations.DbcI) error {
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

func (r *Role) DBDelete(writeDB common_declarations.DbcI) error {
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

func (r *Role) DBSave(writeDB common_declarations.DbcI) error {
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

func (r *Role) DBGet(readDB common_declarations.DbcI) error {
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
