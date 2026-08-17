package role_resource_tiles

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_models"
)

// Init 建表
func Init(writeDB common_declarations.DbcI) {
	err := writeDB.AutoMigrate(&game_models.RoleResourceTile{})
	if err != nil {
		panic(fmt.Sprintf("role_resource_tile auto_migrate failed, err: %s", err.Error()))
	}
}

// DBCreate 创建资源地快照记录
func (rts *RoleResourceTiles) DBCreate(writeDB common_declarations.DbcI) error {
	if len(rts.List) < 1 {
		return nil
	}
	return writeDB.CreateInBatches(rts.List, len(rts.List)).Error()
}

// DBDelete 删除资源地快照记录
func (rts *RoleResourceTiles) DBDelete(writeDB common_declarations.DbcI) error {
	if len(rts.List) < 1 {
		return nil
	}
	return writeDB.Where("role_id = ?", rts.RoleID).Delete(&game_models.RoleResourceTile{}).Error()
}

// DBSave 保存资源地快照记录
func (rts *RoleResourceTiles) DBSave(writeDB common_declarations.DbcI) error {
	if len(rts.List) < 1 {
		return nil
	}
	return writeDB.Save(rts.List).Error()
}

// DBGet 查询资源地快照记录
func (rts *RoleResourceTiles) DBGet(readDB common_declarations.DbcI) error {
	l := make([]*game_models.RoleResourceTile, 0)
	err := readDB.Where("role_id = ?", rts.RoleID).Find(&l).Error()
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if l != nil {
		rts.List = l
		rts.Init()
	}
	return nil
}
