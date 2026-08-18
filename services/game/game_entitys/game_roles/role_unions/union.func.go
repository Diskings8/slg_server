package role_unions

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_union"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

// NewRoleUnions 创建联盟快照子模块
func NewRoleUnions(roleID uint64) *RoleUnions {
	return &RoleUnions{RoleID: roleID}
}

// Init 单行子模块，无需建立索引
func (rus *RoleUnions) Init() {
}

// Copy 深拷贝（副本模式）
func (rus *RoleUnions) Copy(src *RoleUnions) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}
	if err = util_jsons.Unmarshal(b, rus); err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}
}

// UnionID 所属联盟ID（0=无联盟）
func (rus *RoleUnions) UnionID() uint64 {
	if rus.Union == nil {
		return 0
	}
	return rus.Union.UnionID
}

// Position 联盟职位（UnionPosition；无联盟 0）
func (rus *RoleUnions) Position() int32 {
	if rus.Union == nil {
		return 0
	}
	return rus.Union.Position
}

// IsInUnion 是否在联盟中
func (rus *RoleUnions) IsInUnion() bool {
	return rus.Union != nil && rus.Union.UnionID > 0
}

// Set 加入/更新联盟快照：写入 brief 简要信息 + 职位/加入时间
func (rus *RoleUnions) Set(unionID uint64, name string, leaderID uint64, notice string, position int32) {
	now := time.Now().Unix()
	if rus.Union == nil {
		rus.Union = &game_models.RoleUnion{
			RoleID: rus.RoleID,
			JoinUx: now,
		}
	}
	rus.Union.UnionID = unionID
	rus.Union.Name = name
	rus.Union.LeaderID = leaderID
	rus.Union.Notice = notice
	rus.Union.Position = position
	if rus.Union.JoinUx == 0 {
		rus.Union.JoinUx = now
	}
}

// SetPosition 仅更新职位（转让盟主等）
func (rus *RoleUnions) SetPosition(position int32) {
	if rus.Union != nil {
		rus.Union.Position = position
	}
}

// Clear 退出/解散：清空快照
func (rus *RoleUnions) Clear() {
	rus.Union = nil
}

// Format2Pb 转协议对象（RoleUnionInfo：brief + 额外信息）
func (rus *RoleUnions) Format2Pb() *pb_union.RoleUnionInfo {
	if rus.Union == nil || rus.Union.UnionID == 0 {
		return nil
	}
	u := rus.Union
	return &pb_union.RoleUnionInfo{
		Brief: &pb_union.UnionBrief{
			UnionId:     u.UnionID,
			Name:        u.Name,
			LeaderId:    u.LeaderID,
			MemberCount: 0, // 动态值，由 union 聚合侧填充
			Notice:      u.Notice,
		},
		Position: u.Position,
		JoinUx:   u.JoinUx,
	}
}
