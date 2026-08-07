package login_servers_store

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/login/login_models"
)

// ServerStore 区服列表数据访问（login_server 表，common_db_0）
type ServerStore struct {
	db common_declarations.DbcI
}

func NewServerStore(db common_declarations.DbcI) *ServerStore {
	return &ServerStore{db: db}
}

// defaultStore 包级默认 store（单例，login 启动 / 测试 setup 时 Init 设置）
var defaultStore *ServerStore

// Init 初始化默认 store（New + Migrate + 区服种子 + 设单例），login_internals.Init / SetupStores 调用
func Init(db common_declarations.DbcI) error {
	s := NewServerStore(db)
	if err := s.Migrate(); err != nil {
		return err
	}
	if err := s.SeedIfEmpty(); err != nil {
		return err
	}
	defaultStore = s
	return nil
}

// Get 获取包级默认 store（须先 Init；login_logics 直接访问）
func Get() *ServerStore { return defaultStore }

// Migrate 建表（幂等）
func (s *ServerStore) Migrate() error {
	return s.db.AutoMigrate(&login_models.Server{})
}

// ListServers 全部区服（按 id 正序）
func (s *ServerStore) ListServers() ([]*login_models.Server, error) {
	var servers []*login_models.Server
	err := s.db.Order("id ASC").Find(&servers).Error()
	return servers, err
}

// GetServer 按 ID 查区服，不存在返回 nil
func (s *ServerStore) GetServer(serverID uint32) (*login_models.Server, error) {
	var sv login_models.Server
	err := s.db.Where("id = ?", serverID).First(&sv).Error()
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
	if err := s.db.Model(&login_models.Server{}).Count(&count).Error(); err != nil {
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
	}).Error()
}
