package game_models

import "server.slg.com/common/models"

const role_review = "role_review"

// RoleReview 角色审查状态（每角色一行）
type RoleReview struct {
	models.ModelBase
	RoleID   uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_review;comment:角色ID"`
	Chances  int32  `gorm:"column:chances;type:int(11);not null;default:0;comment:审查次数"`
	Exp      int32  `gorm:"column:exp;type:int(11);not null;default:0;comment:审查经验"`
	Level    int32  `gorm:"column:level;type:int(11);not null;default:1;comment:审查等级"`
	LastDate int32  `gorm:"column:last_date;type:int(11);not null;default:0;comment:最后结算日期(YYMMDD,8点切日)"`
}

func (RoleReview) TableName() string { return role_review }
