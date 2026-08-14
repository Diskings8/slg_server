package game_models

import (
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_equip"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/models"
	_ "server.slg.com/common/utils/gormserializers" // 注册 jsonslice 序列化器
)

const role_hero = "role_hero"

// RoleHero 英雄
type RoleHero struct {
	models.ModelBase
	RoleID       uint64 `gorm:"column:role_id;type:bigint(20);not null;index:idx_role_hero;comment:角色ID"` // 普通索引：一个角色可拥有多张英雄（同配置也可多张）
	HeroConfID   int32  `gorm:"column:hero_conf_id;type:int(11);not null;comment:英雄配置ID"`
	StarStage    int32  `gorm:"column:star_stage;type:int(11);not null;default:0;comment:星级(升星消耗同配置英雄卡)"` // 星级（升星消耗同配置英雄卡）
	Level        uint32  `gorm:"column:level;type:int(11) unsigned;not null;default:1;comment:等级"`
	Exp          uint32  `gorm:"column:exp;type:int(11) unsigned;not null;default:0;comment:经验"`
	AttrPoint    uint32  `gorm:"column:attr_point;type:int(11) unsigned;not null;default:0;comment:自由属性点(每10级升级获得)"` // 自由属性点（每10级升级获得）
	Cultivates   []*pb_cultivate.Cultivate  `gorm:"serializer:jsonslice;type:json;not null;comment:养成属性(攻/防/智/移/迁)"`
	EquipSkills  []*pb_skill.Skill      `gorm:"serializer:jsonslice;type:json;not null;comment:已装配技能"`
	EquipWeapons []*pb_equip.Weapon     `gorm:"serializer:jsonslice;type:json;not null;comment:已装配武器"`
	Troops       []*pb_equip.Troop      `gorm:"serializer:jsonslice;type:json;not null;comment:兵种"`
	CurTroopTypeID int32                `gorm:"column:cur_troop_type_id;type:int(11);not null;default:0;comment:当前兵种类型(基础或已转化派生)"` // 当前兵种类型（基础或已转化派生）
	IsLocked     bool                   `gorm:"column:is_locked;type:tinyint(1);not null;default:0;comment:是否锁定"`
	// IsAwakened 英雄是否已觉醒（第三技能槽位解锁条件）
	IsAwakened   bool                   `gorm:"column:is_awakened;type:tinyint(1);not null;default:0;comment:是否已觉醒(第三技能槽位解锁条件)"`
}

func (RoleHero) TableName() string {
	return role_hero
}
