package hero_skills

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
		&game_models.HeroSkill{},
	)
	if err != nil {
		panic(fmt.Sprintf("role hero_skill auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建英雄技能库记录
func (hss *HeroSkills) DBCreate(writeDB common_declarations.DbcI) error {
	if len(hss.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(hss.List, len(hss.List)).Error()
}

// DBDelete 删除英雄技能库
func (hss *HeroSkills) DBDelete(writeDB common_declarations.DbcI) error {
	if len(hss.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", hss.RoleID).Delete(&game_models.HeroSkill{}).Error()
}

// DBSave 保存英雄技能库
func (hss *HeroSkills) DBSave(writeDB common_declarations.DbcI) error {
	if len(hss.List) < 1 {
		return nil
	}
	return writeDB.Save(hss.List).Error()
}

// DBGet 查询英雄技能库
func (hss *HeroSkills) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.HeroSkill, 0)
	err := readDB.Where("role_id = ?", hss.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		hss.List = l
		hss.Init()
	}
	return nil
}
