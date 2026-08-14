package role_heroes

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
		&game_models.RoleHero{},
	)
	if err != nil {
		panic(fmt.Sprintf("role_hero auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建英雄记录
func (hrs *RoleHeroes) DBCreate(writeDB common_declarations.DbcI) error {
	if len(hrs.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(hrs.List, len(hrs.List)).Error()
}

// DBDelete 删除英雄记录
func (hrs *RoleHeroes) DBDelete(writeDB common_declarations.DbcI) error {
	if len(hrs.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", hrs.RoleID).Delete(&game_models.RoleHero{}).Error()
}

// DBDeleteHero 删除单张英雄卡记录（升星消耗等）
func (hrs *RoleHeroes) DBDeleteHero(writeDB common_declarations.DbcI, heroID uint64) error {
	return writeDB.Where("id = ? AND role_id = ?", heroID, hrs.RoleID).Delete(&game_models.RoleHero{}).Error()
}

// DBSave 保存英雄记录
func (hrs *RoleHeroes) DBSave(writeDB common_declarations.DbcI) error {
	if len(hrs.List) < 1 {
		return nil
	}
	return writeDB.Save(hrs.List).Error()
}

// DBGet 查询英雄记录
func (hrs *RoleHeroes) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.RoleHero, 0)
	err := readDB.Where("role_id = ?", hrs.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		hrs.List = l
		hrs.Init()
	}
	return nil
}
