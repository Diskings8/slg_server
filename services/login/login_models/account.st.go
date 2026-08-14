package login_models

import (
	"server.slg.com/common/models"
)

const accountTable = "login_account"

// Account 账户表（跨服全局，落 common_db_0）
//
// account_name 为游戏自己维护的账号身份，全局唯一、与渠道无关。
// 渠道侧的原生账号（SDK UID / openid 等）记录在 login_channel_account 绑定表，不占用本表。
type Account struct {
	models.ModelBase         // ID(account_id) / CreatedAt / UpdatedAt
	AccountName       string `gorm:"column:account_name;type:varchar(64);not null;uniqueIndex:uk_account_name;comment:游戏自有账号名(全局唯一)"`
	PasswordHash      string `gorm:"column:password_hash;type:varchar(64);not null;comment:密码哈希"`
	Status            int32  `gorm:"column:status;type:int(11);not null;default:0;comment:账号状态"`
	LastLoginServerID uint32 `gorm:"column:last_login_server_id;type:int(11);not null;default:0;comment:最后登录区服ID"`
	LastLoginRoleID   uint64 `gorm:"column:last_login_role_id;type:bigint(20);not null;default:0;comment:最后登录角色ID"`
}

func (Account) TableName() string { return accountTable }
