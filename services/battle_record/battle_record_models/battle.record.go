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
	MarchID          uint64   `gorm:"column:march_id;type:bigint(20);not null;comment:行军ID"`
	MarchType        int32    `gorm:"column:march_type;type:int(11);not null;comment:行军类型"`
	MapID            int32    `gorm:"column:map_id;type:int(11);not null;comment:目标地块"` // 目标地块
	AttackerRoleID   uint64   `gorm:"column:attacker_role_id;type:bigint(20);not null;comment:攻击方角色ID"`
	AttackerUnionID  uint64   `gorm:"column:attacker_union_id;type:bigint(20);not null;comment:攻击方联盟ID"`
	DefenderRoleIDs  []uint64 `gorm:"serializer:json;type:json;not null;comment:防守方角色ID列表(可多个)"` // 可多个防守方
	DefenderUnionIDs []uint64 `gorm:"serializer:json;type:json;not null;comment:防守方联盟ID列表"`
	AttackerWin      bool     `gorm:"column:attacker_win;type:tinyint(1);not null;default:0;comment:攻击方是否胜利"`
	IsOccupied       bool     `gorm:"column:is_occupied;type:tinyint(1);not null;default:0;comment:是否占领目标"`
	BuildingDamage   uint64   `gorm:"column:building_damage;type:bigint(20);not null;default:0;comment:建筑耐久损失"`
	Results          []byte   `gorm:"column:results;type:blob;not null;comment:战斗结果(proto.Marshal(BattleResults))"` // proto.Marshal(BattleResults)
	BattleTime       int64    `gorm:"column:battle_time;type:bigint(20);not null;index:idx_bt;comment:战斗时间"`
	ParentID         uint64   `gorm:"column:parent_id;type:bigint(20);not null;default:0;index:idx_parent;comment:父战报ID(0=主战报)"` // 子战报指向主战报；0=主战报
}

func (BattleRecord) TableName() string { return battleRecordTable }
