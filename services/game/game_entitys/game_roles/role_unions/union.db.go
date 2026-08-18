package role_unions

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化数据库表
func Init(writeDB common_declarations.DbcI) {
	if err := writeDB.AutoMigrate(&game_models.RoleUnion{}); err != nil {
		panic(fmt.Sprintf("role_union auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建联盟快照记录（单行）
func (rus *RoleUnions) DBCreate(writeDB common_declarations.DbcI) error {
	if rus.Union == nil {
		return nil
	}
	return writeDB.Create(rus.Union).Error()
}

// DBDelete 删除角色的联盟快照记录
func (rus *RoleUnions) DBDelete(writeDB common_declarations.DbcI) error {
	return writeDB.Where("role_id = ?", rus.RoleID).Delete(&game_models.RoleUnion{}).Error()
}

// DBSave 保存联盟快照记录（全量覆写）
func (rus *RoleUnions) DBSave(writeDB common_declarations.DbcI) error {
	if rus.Union == nil {
		return nil
	}
	return writeDB.Save(rus.Union).Error()
}

// DBGet 从数据库查询联盟快照（单行；无记录保持 nil，由 Join 懒创建）
func (rus *RoleUnions) DBGet(readDB common_declarations.DbcI) error {
	u := &game_models.RoleUnion{}
	err := readDB.Where("role_id = ?", rus.RoleID).Take(u).Error()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	rus.Union = u
	return nil
}
