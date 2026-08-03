package role_buildings

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
		&game_models.RoleBuilding{},
	)
	if err != nil {
		panic(fmt.Sprintf("role_building auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建建筑记录
func (rbs *RoleBuildings) DBCreate(writeDB common_declarations.DbcI) error {
	if len(rbs.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(rbs.List, len(rbs.List)).Error()
}

// DBDelete 删除建筑记录
func (rbs *RoleBuildings) DBDelete(writeDB common_declarations.DbcI) error {
	if len(rbs.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", rbs.RoleID).Delete(&game_models.RoleBuilding{}).Error()
}

// DBSave 保存建筑记录
func (rbs *RoleBuildings) DBSave(writeDB common_declarations.DbcI) error {
	if len(rbs.List) < 1 {
		return nil
	}
	return writeDB.Save(rbs.List).Error()
}

// DBGet 查询建筑记录
func (rbs *RoleBuildings) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.RoleBuilding, 0)
	err := readDB.Where("role_id = ?", rbs.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		rbs.List = l
		rbs.Init()
	}
	return nil
}
