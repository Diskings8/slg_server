package models

type ModelBase struct {
	ID        uint64 `gorm:"primary_key;column:id;type:bigint(20);not null"`
	CreatedAt int64  `gorm:"column:created_at;type:bigint(20);not null"`
	UpdatedAt int64  `gorm:"column:updated_at;type:bigint(20);not null"`
}
