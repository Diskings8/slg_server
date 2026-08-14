package login_models

const channelTable = "login_channel"

// Channel 渠道声明表（落 common_db_0）
//
// 不同渠道在此声明，官方渠道（Mine=0）也只是渠道的一种。
// secret 为第三方渠道 MD5 签名校验密钥，官方渠道可为空。
type Channel struct {
	ChannelType int32  `gorm:"column:channel_type;type:int(11);primary_key;autoIncrement:false;not null;comment:渠道类型"`
	ChannelName string `gorm:"column:channel_name;type:varchar(64);not null;comment:渠道名"`
	Secret      string `gorm:"column:secret;type:varchar(128);not null;default:'';comment:渠道MD5校验密钥"` // 渠道 MD5 校验密钥
	Status      int32  `gorm:"column:status;type:int(11);not null;default:0;comment:渠道状态"`
	CreatedAt   int64  `gorm:"column:created_at;type:bigint(20);not null;comment:创建时间"`
	UpdatedAt   int64  `gorm:"column:updated_at;type:bigint(20);not null;comment:更新时间"`
}

func (Channel) TableName() string { return channelTable }
