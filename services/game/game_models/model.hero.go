package game_models

import (
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_equip"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/models"
)

const role_hero = "role_hero"

// RoleHero 英雄
type RoleHero struct {
	models.ModelBase
	RoleID       uint64 `gorm:"column:role_id;type:bigint(20);not null;index:idx_role_hero"` // 普通索引：一个角色可拥有多张英雄（同配置也可多张）
	HeroConfID   int32  `gorm:"column:hero_conf_id;type:int(11);not null"`
	Level        uint32  `gorm:"column:level;type:int(11) unsigned;not null;default:1"`
	Exp          uint32  `gorm:"column:exp;type:int(11) unsigned;not null;default:0"`
	AttrPoint    uint32  `gorm:"column:attr_point;type:int(11) unsigned;not null;default:0"` // 自由属性点（每10级升级获得）
	Cultivates   []*pb_cultivate.Cultivate  `gorm:"serializer:json;type:json;not null"`
	EquipSkills  []*pb_skill.Skill      `gorm:"serializer:json;type:json;not null"`
	EquipWeapons []*pb_equip.Weapon     `gorm:"serializer:json;type:json;not null"`
	Troops       []*pb_equip.Troop      `gorm:"serializer:json;type:json;not null"`
	CurTroopTypeID int32                `gorm:"column:cur_troop_type_id;type:int(11);not null;default:0"` // 当前兵种类型（基础或已转化派生）
	IsLocked     bool                   `gorm:"column:is_locked;type:tinyint(1);not null;default:0"`
}

func (RoleHero) TableName() string {
	return role_hero
}
