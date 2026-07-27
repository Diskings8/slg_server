package hero_skillcollections

import (
	"fmt"

	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化
func Init(writeDB common_declarations.DbcI) {
	err := writeDB.AutoMigrate(
		&game_models.HeroSkillCollection{},
	)
	if err != nil {
		panic(fmt.Sprintf("hero_skill_collection auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建英雄技能收藏记录
func (hsc *HeroSkillCollections) DBCreate(writeDB common_declarations.DbcI) error {
	if len(hsc.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(hsc.List, len(hsc.List)).Error()
}

// DBDelete 删除英雄技能收藏
func (hsc *HeroSkillCollections) DBDelete(writeDB common_declarations.DbcI) error {
	if len(hsc.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", hsc.RoleID).Delete(&game_models.HeroSkillCollection{}).Error()
}

// DBSave 保存英雄技能收藏
func (hsc *HeroSkillCollections) DBSave(writeDB common_declarations.DbcI) error {
	if len(hsc.List) < 1 {
		return nil
	}
	return writeDB.Save(hsc.List).Error()
}

// DBGet 查询英雄技能收藏
func (hsc *HeroSkillCollections) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.HeroSkillCollection, 0)
	err := readDB.Where("role_id = ?", hsc.RoleID).Find(&l).Error()
	if err != nil {
		return err
	}
	if l != nil {
		hsc.List = l
		hsc.Init()
	}
	return nil
}
