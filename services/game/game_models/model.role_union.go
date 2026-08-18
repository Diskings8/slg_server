package game_models

import "server.slg.com/common/models"

const role_union = "role_union"

// RoleUnion 角色侧联盟快照（1:1 单行，挂 Role 子模块）
//
// 携带 union 的 proto 简要信息（名称/盟主/公告）+ 额外信息（职位/加入时间），
// 角色展示联盟时免 join 联盟表。
type RoleUnion struct {
	models.ModelBase
	RoleID   uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_union,priority:1;comment:角色ID"`
	UnionID  uint64 `gorm:"column:union_id;type:bigint(20);not null;default:0;index;comment:所属联盟ID(0=无)"`
	Position int32  `gorm:"column:position;type:int(11);not null;default:0;comment:联盟职位(UnionPosition)"`
	JoinUx   int64  `gorm:"column:join_ux;type:bigint(20);not null;default:0;comment:加入时间"`
	Name     string `gorm:"column:name;type:varchar(64);not null;default:'';comment:联盟名(快照)"`
	LeaderID uint64 `gorm:"column:leader_id;type:bigint(20);not null;default:0;comment:盟主角色ID(快照)"`
	Notice   string `gorm:"column:notice;type:varchar(255);not null;default:'';comment:公告(快照)"`
}

func (RoleUnion) TableName() string {
	return role_union
}
