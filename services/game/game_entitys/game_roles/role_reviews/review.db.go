package role_reviews

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 初始化数据库表
func Init(writeDB common_declarations.DbcI) {
	if err := writeDB.AutoMigrate(&game_models.RoleReview{}); err != nil {
		panic(fmt.Sprintf("role_review auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建审查记录（单行）
func (rrs *RoleReviews) DBCreate(writeDB common_declarations.DbcI) error {
	if rrs.Review == nil {
		return nil
	}
	return writeDB.Create(rrs.Review).Error()
}

// DBDelete 删除角色的审查记录
func (rrs *RoleReviews) DBDelete(writeDB common_declarations.DbcI) error {
	return writeDB.Where("role_id = ?", rrs.RoleID).Delete(&game_models.RoleReview{}).Error()
}

// DBSave 保存审查记录
func (rrs *RoleReviews) DBSave(writeDB common_declarations.DbcI) error {
	if rrs.Review == nil {
		return nil
	}
	return writeDB.Save(rrs.Review).Error()
}

// DBGet 加载审查记录（无记录保持默认值，不报错）
func (rrs *RoleReviews) DBGet(readDB common_declarations.DbcI) error {
	if readDB == nil || rrs.Review == nil {
		return nil
	}
	r := &game_models.RoleReview{}
	if err := readDB.Where("role_id = ?", rrs.RoleID).Take(r).Error(); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	rrs.Review = r
	return nil
}
