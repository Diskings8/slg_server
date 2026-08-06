package login_models

const channelAccountTable = "login_channel_account"

// ChannelAccount 渠道账号绑定表（落 common_db_0）
//
// 记录渠道侧原生账号（channel_account）关联到游戏账户（account_id）。
// 一个账户可绑多个渠道（account_id 索引非唯一）；同一渠道账号只映射到一个账户（联合唯一）。
// auth_info 记录该渠道登录时提供的认证信息（如第三方 MD5 签名），留痕 / 后续可校验。
type ChannelAccount struct {
	ID             uint64 `gorm:"column:id;type:bigint(20);primary_key;not null"` // 应用层雪花 ID 生成
	AccountID      uint64 `gorm:"column:account_id;type:bigint(20);not null;index:idx_ca_account"`
	ChannelType    int32  `gorm:"column:channel_type;type:int(11);not null;uniqueIndex:uk_channel_account,priority:1"`
	ChannelAccount string `gorm:"column:channel_account;type:varchar(64);not null;uniqueIndex:uk_channel_account,priority:2"`
	AuthInfo       string `gorm:"column:auth_info;type:varchar(256);not null;default:''"` // 渠道登录时提供的认证信息
	CreatedAt      int64  `gorm:"column:created_at;type:bigint(20);not null"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint(20);not null"`
}

func (ChannelAccount) TableName() string { return channelAccountTable }
