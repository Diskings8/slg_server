package login_channels

import (
	"errors"

	"gorm.io/gorm"
	"server.slg.com/services/login/login_models"
)

// ChannelStore 渠道声明数据访问（login_channel 表，common_db_0）
type ChannelStore struct {
	db *gorm.DB
}

func NewChannelStore(db *gorm.DB) *ChannelStore {
	return &ChannelStore{db: db}
}

// Migrate 建表（幂等）
func (s *ChannelStore) Migrate() error {
	return s.db.AutoMigrate(&login_models.Channel{})
}

// GetChannel 按渠道类型查声明，未声明返回 nil
func (s *ChannelStore) GetChannel(channelType int32) (*login_models.Channel, error) {
	var ch login_models.Channel
	err := s.db.Where("channel_type = ?", channelType).First(&ch).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ch, nil
}

// SeedDefault 幂等插入官方渠道（Mine=0）。其他渠道后续在 login_channel 表中声明。
func (s *ChannelStore) SeedDefault() error {
	var count int64
	if err := s.db.Model(&login_models.Channel{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return s.db.Create(&login_models.Channel{
		ChannelType: 0,
		ChannelName: "官方渠道",
		Status:      0,
	}).Error
}
