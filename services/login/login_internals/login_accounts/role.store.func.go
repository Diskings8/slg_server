package login_accounts

import (
	"errors"

	"gorm.io/gorm"
	"server.slg.com/services/login/login_models"
)

// CreateRole 写入账号-角色映射；撞唯一索引返回 ErrRoleExists
func (s *AccountStore) CreateRole(r *login_models.Role) error {
	if err := s.db.Create(r).Error(); err != nil {
		if isDuplicateKey(err) {
			return ErrRoleExists
		}
		return err
	}
	return nil
}

// GetRoleByAccountServer 按账号 + 区服查角色（每账号每服最多一个），不存在返回 nil
func (s *AccountStore) GetRoleByAccountServer(accountID uint64, serverID uint32) (*login_models.Role, error) {
	var r login_models.Role
	err := s.db.Where("account_id = ? AND server_id = ?", accountID, serverID).First(&r).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetRolesByAccount 查账号名下所有角色（跨服，按创建时间正序）
func (s *AccountStore) GetRolesByAccount(accountID uint64) ([]*login_models.Role, error) {
	var roles []*login_models.Role
	err := s.db.Where("account_id = ?", accountID).Order("created_at ASC").Find(&roles).Error()
	return roles, err
}

// GetRoleByName 按区服 + 角色名查（服内角色名唯一，用于建角前查重），不存在返回 nil
func (s *AccountStore) GetRoleByName(serverID uint32, roleName string) (*login_models.Role, error) {
	var r login_models.Role
	err := s.db.Where("server_id = ? AND role_name = ?", serverID, roleName).First(&r).Error()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}
