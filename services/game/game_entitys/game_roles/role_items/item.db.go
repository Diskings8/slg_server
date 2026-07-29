package role_items

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化数据库表
func Init(writeDB common_declarations.DbcI) {
	err := writeDB.AutoMigrate(
		&game_models.RoleItem{},
	)
	if err != nil {
		panic(fmt.Sprintf("role_item auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建道具记录
func (ris *RoleItems) DBCreate(writeDB common_declarations.DbcI) error {
	if len(ris.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(ris.List, len(ris.List)).Error()
}

// DBDelete 删除角色的所有道具记录
func (ris *RoleItems) DBDelete(writeDB common_declarations.DbcI) error {
	if len(ris.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", ris.RoleID).Delete(&game_models.RoleItem{}).Error()
}

// DBSave 保存道具记录
func (ris *RoleItems) DBSave(writeDB common_declarations.DbcI) error {
	if len(ris.List) < 1 {
		return nil
	}
	return writeDB.Save(ris.List).Error()
}

// DBGet 从数据库查询角色道具
func (ris *RoleItems) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.RoleItem, 0)
	err := readDB.Where("role_id = ?", ris.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		ris.List = l
		ris.Init()
	}
	return nil
}
