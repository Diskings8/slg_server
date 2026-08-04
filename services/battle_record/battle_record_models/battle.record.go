package battle_record_models

import (
	"server.slg.com/common/models"
)

const battleRecordTable = "battle_record"

// BattleRecord 战报主表
//
// 只存记录本身，查询走 battle_record_tag 索引表（避免多维 × 攻守多个索引的写入放大）。
// Results 为 proto.Marshal(pb_battle.BattleResults) 的二进制。
type BattleRecord struct {
	models.ModelBase          // ID / CreatedAt / UpdatedAt
	MarchID          uint64   `gorm:"column:march_id;type:bigint(20);not null"`
	MarchType        int32    `gorm:"column:march_type;type:int(11);not null"`
	MapID            int32    `gorm:"column:map_id;type:int(11);not null"` // 目标地块
	AttackerRoleID   uint64   `gorm:"column:attacker_role_id;type:bigint(20);not null"`
	AttackerUnionID  uint64   `gorm:"column:attacker_union_id;type:bigint(20);not null"`
	DefenderRoleIDs  []uint64 `gorm:"serializer:json;type:json;not null"` // 可多个防守方
	DefenderUnionIDs []uint64 `gorm:"serializer:json;type:json;not null"`
	AttackerWin      bool     `gorm:"column:attacker_win;type:tinyint(1);not null;default:0"`
	IsOccupied       bool     `gorm:"column:is_occupied;type:tinyint(1);not null;default:0"`
	BuildingDamage   uint64   `gorm:"column:building_damage;type:bigint(20);not null;default:0"`
	Results          []byte   `gorm:"column:results;type:blob;not null"` // proto.Marshal(BattleResults)
	BattleTime       int64    `gorm:"column:battle_time;type:bigint(20);not null;index:idx_bt"`
}

func (BattleRecord) TableName() string { return battleRecordTable }
