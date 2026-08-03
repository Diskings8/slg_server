package role_formations

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化
func Init(writeDB common_declarations.DbcI) {
	err := writeDB.AutoMigrate(
		&game_models.RoleFormation{},
	)
	if err != nil {
		panic(fmt.Sprintf("role_formation auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建编队记录
func (rfs *RoleFormations) DBCreate(writeDB common_declarations.DbcI) error {
	if len(rfs.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(rfs.List, len(rfs.List)).Error()
}

// DBDelete 删除编队记录
func (rfs *RoleFormations) DBDelete(writeDB common_declarations.DbcI) error {
	if len(rfs.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", rfs.RoleID).Delete(&game_models.RoleFormation{}).Error()
}

// DBSave 保存编队记录
func (rfs *RoleFormations) DBSave(writeDB common_declarations.DbcI) error {
	if len(rfs.List) < 1 {
		return nil
	}
	return writeDB.Save(rfs.List).Error()
}

// DBGet 查询编队记录
func (rfs *RoleFormations) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.RoleFormation, 0)
	err := readDB.Where("role_id = ?", rfs.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		rfs.List = l
		rfs.Init()
	}
	return nil
}
