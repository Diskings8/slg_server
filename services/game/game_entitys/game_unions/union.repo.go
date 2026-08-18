// Package game_unions 联盟聚合实体（game 节点持有，gorm 持久化）
package game_unions

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/game/game_models"
)

// Repo 联盟持久化接口（默认 gorm 实现；测试可注入替换）
type Repo interface {
	Get(unionID uint64) (*game_models.Union, error) // 不存在返回 (nil, nil)
	Create(u *game_models.Union) (uint64, error)
	Save(u *game_models.Union) error
	Delete(unionID uint64) error
	ExistsName(name string) (bool, error)
}

var repo Repo = &gormRepo{}

// SetRepo 注入联盟持久化实现（测试用）
func SetRepo(r Repo) { repo = r }

// GetRepo 获取联盟持久化
func GetRepo() Repo { return repo }

// Init 建表
func Init(writeDB common_declarations.DbcI) {
	if err := writeDB.AutoMigrate(&game_models.Union{}); err != nil {
		panic(fmt.Sprintf("union auto_migrate failed, err: %s", err.Error()))
	}
}

// gormRepo 默认 gorm 实现
type gormRepo struct{}

func (gormRepo) Get(unionID uint64) (*game_models.Union, error) {
	u := &game_models.Union{}
	err := dbconn.GetWriteDbConn().Where("id = ?", unionID).Take(u).Error()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (gormRepo) Create(u *game_models.Union) (uint64, error) {
	u.ID = snowflakes.GenUUID()
	if err := dbconn.GetWriteDbConn().Create(u).Error(); err != nil {
		return 0, err
	}
	return u.ID, nil
}

func (gormRepo) Save(u *game_models.Union) error {
	return dbconn.GetWriteDbConn().Save(u).Error()
}

func (gormRepo) Delete(unionID uint64) error {
	return dbconn.GetWriteDbConn().Delete(&game_models.Union{}, unionID).Error()
}

func (gormRepo) ExistsName(name string) (bool, error) {
	var count int64
	err := dbconn.GetWriteDbConn().Model(&game_models.Union{}).Where("name = ?", name).Count(&count).Error()
	return count > 0, err
}
