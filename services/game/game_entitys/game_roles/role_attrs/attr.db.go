package role_attrs

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化数据库表
func Init(writeDB common_declarations.DbcI) {
	if err := writeDB.AutoMigrate(&game_models.RoleAttr{}); err != nil {
		panic(fmt.Sprintf("role_attr auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建属性记录（单行）
func (ras *RoleAttrs) DBCreate(writeDB common_declarations.DbcI) error {
	if ras.Attr == nil {
		return nil
	}
	return writeDB.Create(ras.Attr).Error()
}

// DBDelete 删除角色的属性记录
func (ras *RoleAttrs) DBDelete(writeDB common_declarations.DbcI) error {
	return writeDB.Where("role_id = ?", ras.RoleID).Delete(&game_models.RoleAttr{}).Error()
}

// DBSave 保存属性记录（全量覆写）
func (ras *RoleAttrs) DBSave(writeDB common_declarations.DbcI) error {
	if ras.Attr == nil {
		return nil
	}
	return writeDB.Save(ras.Attr).Error()
}

// DBGet 从数据库查询角色属性（单行；无记录保持 nil，由 Ensure 懒创建）
func (ras *RoleAttrs) DBGet(readDB common_declarations.DbcI) error {
	attr := &game_models.RoleAttr{}
	err := readDB.Where("role_id = ?", ras.RoleID).Take(attr).Error()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	ras.Attr = attr
	return nil
}
