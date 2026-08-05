package role_recruits

import (
	"server.slg.com/services/game/game_models"
)

// RoleRecruits 角色抽卡数据（每角色单行整包，Data 存全部池状态）
type RoleRecruits struct {
	RoleID uint64                      `json:"role_id"`
	Data   game_models.RoleRecruitData `json:"data"`
}

// NewRoleRecruits 创建角色抽卡数据
func NewRoleRecruits(roleID uint64) *RoleRecruits {
	return &RoleRecruits{
		RoleID: roleID,
		Data:   game_models.NewRecruitData(),
	}
}
