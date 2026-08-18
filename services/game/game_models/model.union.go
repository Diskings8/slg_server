package game_models

import (
	"server.slg.com/api/protocol/pb/pb_union"
	"server.slg.com/common/models"
)

// 表名用 slg_union（union 是 SQL 关键字，避免歧义）
const slg_union = "slg_union"

// Union 联盟聚合（game 节点持有，完整成员列表）
type Union struct {
	models.ModelBase
	Name     string                  `gorm:"column:name;type:varchar(64);not null;uniqueIndex;comment:联盟名(服内唯一)"`
	LeaderID uint64                  `gorm:"column:leader_id;type:bigint(20);not null;default:0;comment:盟主角色ID"`
	Notice   string                  `gorm:"column:notice;type:varchar(255);not null;default:'';comment:公告"`
	Members  []*pb_union.UnionMember `gorm:"serializer:jsonslice;type:json;not null;comment:成员列表(角色+职位+简要)"`
}

func (Union) TableName() string {
	return slg_union
}
