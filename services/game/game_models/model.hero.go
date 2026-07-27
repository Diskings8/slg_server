package game_models

import (
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/common/models"
)

const role_hero = "role_hero"

// RoleHero 英雄
type RoleHero struct {
	models.ModelBase
	RoleID       uint64 `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_hero_skill"`
	HeroConfID   int32
	Level        uint32
	Exp          uint32
	Cultivates   []*pb_hero.Cultivate
	EquipSkills  []*pb_hero.Skill
	EquipWeapons []*pb_hero.Weapon
	TroopTypes   []*pb_hero.TroopType
	IsLocked     bool
}

func (RoleHero) TableName() string {
	return role_hero
}
