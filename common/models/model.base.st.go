package models

type ModelBase struct {
	ID        uint64 `gorm:"primary_key;column:id;type:bigint(20);not null;comment:主键ID"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint(20);not null;comment:创建时间"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint(20);not null;comment:更新时间"`
}
