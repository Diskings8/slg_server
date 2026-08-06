package login_servers

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"server.slg.com/services/login/login_models"
)

// ServerStore 区服列表数据访问（login_server 表，common_db_0）
type ServerStore struct {
	db *gorm.DB
}

func NewServerStore(db *gorm.DB) *ServerStore {
	return &ServerStore{db: db}
}

// Migrate 建表（幂等）
func (s *ServerStore) Migrate() error {
	return s.db.AutoMigrate(&login_models.Server{})
}

// ListServers 全部区服（按 id 正序）
func (s *ServerStore) ListServers() ([]*login_models.Server, error) {
	var servers []*login_models.Server
	err := s.db.Order("id ASC").Find(&servers).Error
	return servers, err
}

// GetServer 按 ID 查区服，不存在返回 nil
func (s *ServerStore) GetServer(serverID uint32) (*login_models.Server, error) {
	var sv login_models.Server
	err := s.db.Where("id = ?", serverID).First(&sv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sv, nil
}

// SeedIfEmpty 表空时插入默认区服（幂等种子，本地起 login 即可用）
func (s *ServerStore) SeedIfEmpty() error {
	var count int64
	if err := s.db.Model(&login_models.Server{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	now := time.Now().Unix()
	return s.db.Create(&login_models.Server{
		ID:         1,
		ServerName: "S1",
		Status:     0,
		OpenTime:   now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}).Error
}
