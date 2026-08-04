package battle_record_models

const battleRecordTagTable = "battle_record_tag"

// BattleRecordTag 战报查询索引表
//
// 一条战报按攻守双方 × 三维（角色/联盟/地块）生成多行，单复合索引 (tag_type, tag_id, battle_time)
// 覆盖所有维度查询，避免主表多索引。
// 内部索引行，主键用自增（无需雪花全局唯一 ID）。
type BattleRecordTag struct {
	ID             uint64 `gorm:"primaryKey;column:id;autoIncrement"`
	CreatedAt      int64  `gorm:"column:created_at;type:bigint(20);not null"`
	UpdatedAt      int64  `gorm:"column:updated_at;type:bigint(20);not null"`
	TagType        int32  `gorm:"column:tag_type;type:int(11);not null;index:idx_tag_query,priority:1"` // 1=role 2=union 3=tile
	TagID          uint64 `gorm:"column:tag_id;type:bigint(20);not null;index:idx_tag_query,priority:2"`
	BattleRecordID uint64 `gorm:"column:battle_record_id;type:bigint(20);not null"`
	BattleTime     int64  `gorm:"column:battle_time;type:bigint(20);not null;index:idx_tag_query,priority:3;index:idx_tag_time"` // 复合查询索引 + 清理索引
}

func (BattleRecordTag) TableName() string { return battleRecordTagTable }
