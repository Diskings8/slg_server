package game_models

const TableNameRoleHero = "role_heroes"

// RoleHero mapped from table <role_heroes>
type RoleHero struct {
	RoleID    uint64 `gorm:"column:role_id;type:bigint unsigned;primaryKey;autoIncrement:false;comment:角色 id" json:"role_id"` // 角色 id
	ID        uint32 `gorm:"column:id;type:int unsigned;primaryKey;autoIncrement:false;comment:英雄唯一id" json:"id"`           // 英雄唯一id
	Level     uint32 `gorm:"column:level;type:int unsigned;not null;comment:英雄等级" json:"level"`                             // 英雄等级
	StarStage uint32 `gorm:"column:star_stage;type:int unsigned;not null;comment:星阶数" json:"star_stage"`                     // 星阶数
	CityID    uint32 `gorm:"column:city_id;type:int unsigned;not null;comment:上阵城市" json:"city_id"`                         // 上阵城市
	TeamID    uint32 `gorm:"column:team_id;type:int unsigned;not null;comment:编队号" json:"team_id"`                           // 编队号

}

// TableName RoleHero's table name
func (*RoleHero) TableName() string {
	return TableNameRoleHero
}
