package role_recruits

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/models"
	"server.slg.com/services/game/game_models"
)

// Init 初始化
func Init(writeDB common_declarations.DbcI) {
	err := writeDB.AutoMigrate(&game_models.RoleRecruit{})
	if err != nil {
		panic(fmt.Sprintf("role_recruit auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建（每角色一行，Save 幂等 upsert）
func (rrs *RoleRecruits) DBCreate(writeDB common_declarations.DbcI) error {
	return rrs.DBSave(writeDB)
}

// DBDelete 删除
func (rrs *RoleRecruits) DBDelete(writeDB common_declarations.DbcI) error {
	return writeDB.Where("role_id = ?", rrs.RoleID).Delete(&game_models.RoleRecruit{}).Error()
}

// DBSave 保存整包数据（以角色ID作为主键，Save 命中同一行）
func (rrs *RoleRecruits) DBSave(writeDB common_declarations.DbcI) error {
	if rrs.Data.Pools == nil {
		rrs.Data.Pools = make(map[uint32]*game_models.RecruitPool)
	}
	now := time.Now().Unix()
	m := &game_models.RoleRecruit{
		ModelBase: models.ModelBase{
			ID:        rrs.RoleID,
			CreatedAt: now,
			UpdatedAt: now,
		},
		RoleID: rrs.RoleID,
		Data:   rrs.Data,
	}
	return writeDB.Save(m).Error()
}

// DBGet 查询（无记录时保留默认空数据）
func (rrs *RoleRecruits) DBGet(readDB common_declarations.DbcI) error {
	m := &game_models.RoleRecruit{}
	err := readDB.Where("role_id = ?", rrs.RoleID).Take(m).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err == nil {
		rrs.Data = m.Data
		if rrs.Data.Pools == nil {
			rrs.Data.Pools = make(map[uint32]*game_models.RecruitPool)
		}
	}
	return nil
}
