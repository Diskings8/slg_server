package login_accounts

import (
	"errors"
	"strings"

	"gorm.io/gorm"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_models"
)

var (
	// ErrAccountExists 账号已存在（account_name 全局唯一）
	ErrAccountExists = errors.New("account already exists")
	// ErrRoleExists 角色已存在（撞唯一索引：服内角色名唯一 / 每账号每服一个角色）
	ErrRoleExists = errors.New("role already exists")
	// ErrChannelExists 渠道账号已被绑定（同一渠道账号只映射到一个账户）
	ErrChannelExists = errors.New("channel account already bound")
)

// isDuplicateKey 跨驱动识别唯一索引冲突。
//
// MySQL 驱动会把 1062 翻译成 gorm.ErrDuplicatedKey；sqlite 驱动不翻译，需按错误文本兜底。
func isDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "UNIQUE constraint failed")
}

// AccountStore 账户/渠道绑定/角色映射数据访问（login_account + login_channel_account + login_role，均在 common_db_0）
type AccountStore struct {
	db common_declarations.DbcI
}

func NewAccountStore(db common_declarations.DbcI) *AccountStore {
	return &AccountStore{db: db}
}

// Migrate 建表（AutoMigrate 幂等，索引由 model tag 声明）
func (s *AccountStore) Migrate() error {
	return s.db.AutoMigrate(&login_models.Account{}, &login_models.ChannelAccount{}, &login_models.Role{})
}

// CreateAccountWithChannel 创建账户 + 首个渠道绑定（事务，二者同生共灭）
//
// account_id 由雪花算法生成，绑定行的 account_id 同步填充；渠道账号已绑定 → ErrChannelExists。
func (s *AccountStore) CreateAccountWithChannel(acc *login_models.Account, binding *login_models.ChannelAccount) error {
	if acc.ID == 0 {
		acc.ID = snowflakes.GenUUID()
	}
	if binding.ID == 0 {
		binding.ID = snowflakes.GenUUID()
	}
	binding.AccountID = acc.ID
	return s.db.Transaction(func(tx common_declarations.DbcI) error {
		if err := tx.Create(acc).Error(); err != nil {
			if isDuplicateKey(err) {
				return ErrAccountExists
			}
			return err
		}
		if err := tx.Create(binding).Error(); err != nil {
			if isDuplicateKey(err) {
				return ErrChannelExists
			}
			return err
		}
		return nil
	})
}

// GetAccountByName 按账号名查询（游戏侧全局唯一），不存在返回 nil
func (s *AccountStore) GetAccountByName(name string) (*login_models.Account, error) {
	var acc login_models.Account
	err := s.db.Where("account_name = ?", name).First(&acc).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// GetAccountByID 按账号 ID 查询，不存在返回 nil
func (s *AccountStore) GetAccountByID(id uint64) (*login_models.Account, error) {
	var acc login_models.Account
	err := s.db.Where("id = ?", id).First(&acc).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &acc, nil
}

// UpdateLastLogin 更新账号最近登录的区服/角色（供角色列表 last_use 填充）
func (s *AccountStore) UpdateLastLogin(accountID uint64, serverID uint32, roleID uint64) error {
	return s.db.Model(&login_models.Account{}).
		Where("id = ?", accountID).
		Updates(map[string]interface{}{
			"last_login_server_id": serverID,
			"last_login_role_id":   roleID,
		}).Error()
}

// CreateChannel 写入渠道绑定；渠道账号已绑定 → ErrChannelExists（自动绑定用）
func (s *AccountStore) CreateChannel(binding *login_models.ChannelAccount) error {
	if binding.ID == 0 {
		binding.ID = snowflakes.GenUUID()
	}
	if err := s.db.Create(binding).Error(); err != nil {
		if isDuplicateKey(err) {
			return ErrChannelExists
		}
		return err
	}
	return nil
}

// GetChannel 按渠道 + 渠道账号查绑定，不存在返回 nil
func (s *AccountStore) GetChannel(channelType int32, channelAccount string) (*login_models.ChannelAccount, error) {
	var b login_models.ChannelAccount
	err := s.db.Where("channel_type = ? AND channel_account = ?", channelType, channelAccount).First(&b).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

// UpdateChannelAuthInfo 刷新渠道绑定的认证信息（登录时留痕）
func (s *AccountStore) UpdateChannelAuthInfo(id uint64, authInfo string) error {
	return s.db.Model(&login_models.ChannelAccount{}).
		Where("id = ?", id).
		Update("auth_info", authInfo).Error()
}
