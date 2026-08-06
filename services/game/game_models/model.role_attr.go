package game_models

import (
	"server.slg.com/common/models"
)

const role_attr = "role_attr"

// RoleAttr 角色属性（RoleID 1:1 单行）
type RoleAttr struct {
	models.ModelBase
	RoleID     uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_attr"` // 角色ID
	RoleName   string `gorm:"column:role_name;type:varchar(64);not null;default:''"`             // 角色名
	ServerID   uint32 `gorm:"column:server_id;type:int(11) unsigned;not null;default:0"`         // 所在区服
	CreateAt   int64  `gorm:"column:create_at;type:bigint(20);not null;default:0"`               // 创建时间戳（首次登录记录）
	CreateIP   string `gorm:"column:create_ip;type:varchar(32);not null;default:''"`             // 创建IP
	LoginAt    int64  `gorm:"column:login_at;type:bigint(20);not null;default:0"`                // 最后登录时间戳
	LogoutAt   int64  `gorm:"column:logout_at;type:bigint(20);not null;default:0"`               // 最后登出时间戳
	LoginDays  uint32 `gorm:"column:login_days;type:int(11) unsigned;not null;default:0"`        // 累计登录天数
	OnlineTime uint64 `gorm:"column:online_time;type:bigint(20) unsigned;not null;default:0"`    // 累计在线时长(秒)
	VipLevel   int32  `gorm:"column:vip_level;type:int(11);not null;default:1"`                  // VIP等级（默认1）
	VipExp     int32  `gorm:"column:vip_exp;type:int(11);not null;default:0"`                    // VIP经验
	VipEndTime int64  `gorm:"column:vip_end_time;type:bigint(20);not null;default:0"`            // VIP到期时间戳
}

func (RoleAttr) TableName() string {
	return role_attr
}
