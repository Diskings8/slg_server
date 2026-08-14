package role_reviews

import (
	"server.slg.com/services/game/game_models"
)

// TaskReward 审查任务奖励（道具）
type TaskReward struct {
	ItemID int32 `json:"item_id"`
	Count  int32 `json:"count"`
}

// ReviewTask 待处理审查任务（内存态，选择执行后消耗）
type ReviewTask struct {
	TaskID  int64        `json:"task_id"`
	Type    int32        `json:"type"` // 事件类型（map_events.EventType：采集/打怪/寻宝）
	Rewards []TaskReward `json:"rewards"`
}

// RoleReviews 角色审查子模块（RoleID 1:1 单行）
type RoleReviews struct {
	RoleID uint64 `json:"role_id"`
	Review *game_models.RoleReview `json:"review,omitempty"`

	Pending    []ReviewTask `json:"-"` // 待处理任务（内存态，不持久化）
	nextTaskID int64        `json:"-"`
}

// NewRoleReviews 构造审查子模块（默认等级 1）
func NewRoleReviews(roleID uint64) *RoleReviews {
	return &RoleReviews{
		RoleID: roleID,
		Review: &game_models.RoleReview{RoleID: roleID, Level: 1},
	}
}
