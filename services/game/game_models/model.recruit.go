package game_models

import (
	"server.slg.com/common/models"
)

const role_recruit = "role_recruit"

// RecruitPool 单个抽卡池的玩家状态
type RecruitPool struct {
	ID         uint32 `json:"id"`          // 抽卡池配置ID
	AllTimes   uint32 `json:"all_times"`   // 总次数
	GuardTimes uint32 `json:"guard_times"` // 保底累计
	Wish       uint32 `json:"wish"`        // 心愿进度
	ChooseHero int32  `json:"choose_hero"` // 心愿英雄配置ID；0=未设置

	WindowID  int64  `json:"window_id"`  // 当前免费/半价窗口ID（每天 0/12 点切换，跨窗口重置 FreeTimes/HalfTimes）
	FreeTimes uint32 `json:"free_times"` // 本窗口免费已用次数
	HalfTimes uint32 `json:"half_times"` // 本窗口半价已用次数
}

// RoleRecruitData 抽卡模块整包数据（Data 列 JSON 序列化）
type RoleRecruitData struct {
	Pools map[uint32]*RecruitPool `json:"pools"`
}

// NewRecruitData 构造抽卡数据
func NewRecruitData() RoleRecruitData {
	return RoleRecruitData{Pools: make(map[uint32]*RecruitPool)}
}

// RoleRecruit 角色抽卡持久化（每角色一行）
type RoleRecruit struct {
	models.ModelBase
	RoleID uint64          `gorm:"column:role_id;type:bigint(20);not null;uniqueIndex:idx_role_recruit;comment:角色ID"` // 角色ID
	Data   RoleRecruitData `gorm:"column:data;serializer:jsonslice;type:json;not null;comment:抽卡数据(整包JSON)"`             // 抽卡数据（整包）
}

func (RoleRecruit) TableName() string {
	return role_recruit
}
