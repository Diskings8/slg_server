package cultivate_costs

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
		&game_models.CultivateCost{},
	)
	if err != nil {
		panic(fmt.Sprintf("cultivate_cost auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建养成消耗记录
func (ccs *CultivateCosts) DBCreate(writeDB common_declarations.DbcI) error {
	if len(ccs.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(ccs.List, len(ccs.List)).Error()
}

// DBDelete 删除养成消耗记录
func (ccs *CultivateCosts) DBDelete(writeDB common_declarations.DbcI) error {
	if len(ccs.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", ccs.RoleID).Delete(&game_models.CultivateCost{}).Error()
}

// DBSave 保存养成消耗记录
func (ccs *CultivateCosts) DBSave(writeDB common_declarations.DbcI) error {
	if len(ccs.List) < 1 {
		return nil
	}
	return writeDB.Save(ccs.List).Error()
}

// DBGet 查询养成消耗记录
func (ccs *CultivateCosts) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.CultivateCost, 0)
	err := readDB.Where("role_id = ?", ccs.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		ccs.List = l
		ccs.Init()
	}
	return nil
}
