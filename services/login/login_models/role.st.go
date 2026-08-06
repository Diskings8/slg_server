package login_models

const roleTable = "login_role"

// Role 账号-角色映射表（跨服全局，落 common_db_0）
//
// 主键即游戏内角色 id（login 用雪花算法分配，与 game 侧 CreateRole 的 roleId 同值）。
// 游戏内的角色数据（出生点/主城/属性等）在 game_db_0，这里只存"账号 × 区服 × 角色"映射，
// 供登录构建角色列表、进入区服时校验归属。
type Role struct {
	RoleID    uint64 `gorm:"column:role_id;type:bigint(20);primary_key;not null"` // 游戏内角色 id（login 分配，全局唯一）
	AccountID uint64 `gorm:"column:account_id;type:bigint(20);not null;uniqueIndex:uk_account_server,priority:1;index:idx_account"`
	ServerID  uint32 `gorm:"column:server_id;type:int(11);not null;uniqueIndex:uk_account_server,priority:2;uniqueIndex:uk_server_name,priority:1"`
	RoleName  string `gorm:"column:role_name;type:varchar(64);not null;uniqueIndex:uk_server_name,priority:2"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint(20);not null"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint(20);not null"`
}

func (Role) TableName() string { return roleTable }
