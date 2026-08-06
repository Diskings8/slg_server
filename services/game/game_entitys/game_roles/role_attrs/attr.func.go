package role_attrs

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_attr"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewRoleAttrs(roleID uint64) *RoleAttrs {
	return &RoleAttrs{
		RoleID: roleID,
	}
}

func (ras *RoleAttrs) Init() {
	// 单行子模块，无需建立索引
}

func (ras *RoleAttrs) Copy(src *RoleAttrs) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}
	if err = util_jsons.Unmarshal(b, ras); err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}
}

// Ensure 获取 Attr，懒创建默认值（VipLevel=1）
func (ras *RoleAttrs) Ensure() *game_models.RoleAttr {
	if ras.Attr == nil {
		ras.Attr = &game_models.RoleAttr{
			RoleID:   ras.RoleID,
			VipLevel: 1,
		}
	}
	return ras.Attr
}

// UpdateLogin 登录统计：跨日登录天数+1；首次登录记录 CreateAt；更新 LoginAt
func (ras *RoleAttrs) UpdateLogin() {
	attr := ras.Ensure()
	now := time.Now().Unix()
	if attr.CreateAt == 0 {
		attr.CreateAt = now
	}
	if attr.LoginAt == 0 || dayStart(now) != dayStart(attr.LoginAt) {
		attr.LoginDays++
	}
	attr.LoginAt = now
}

// UpdateLogout 登出统计：累计在线时长，更新 LogoutAt
func (ras *RoleAttrs) UpdateLogout() {
	attr := ras.Ensure()
	now := time.Now().Unix()
	if attr.LoginAt > 0 && now > attr.LoginAt {
		attr.OnlineTime += uint64(now - attr.LoginAt)
	}
	attr.LogoutAt = now
}

// VipLevelEffective 有效VIP等级：已过期（VipEndTime>0 且早于当前）返回 0
func (ras *RoleAttrs) VipLevelEffective() int32 {
	attr := ras.Ensure()
	if attr.VipEndTime > 0 && attr.VipEndTime < time.Now().Unix() {
		return 0
	}
	return attr.VipLevel
}

// Format2Pb 转为协议对象
func (ras *RoleAttrs) Format2Pb() *pb_attr.RoleAttr {
	attr := ras.Ensure()
	return &pb_attr.RoleAttr{
		ServerId:   attr.ServerID,
		CreateAt:   attr.CreateAt,
		CreateIp:   attr.CreateIP,
		LoginAt:    attr.LoginAt,
		LogoutAt:   attr.LogoutAt,
		LoginDays:  attr.LoginDays,
		OnlineTime: attr.OnlineTime,
		VipLevel:   ras.VipLevelEffective(),
		VipExp:     attr.VipExp,
		VipEndTime: attr.VipEndTime,
	}
}

// FillRoleSimpleInfo 填充角色简略信息（role_name / server_id / vip_level），供 account 登录流调用
func (ras *RoleAttrs) FillRoleSimpleInfo(simple *pb_role.RoleSimpleInfo) {
	if simple == nil {
		return
	}
	attr := ras.Ensure()
	simple.RoleName = attr.RoleName
	simple.ServerId = attr.ServerID
	simple.VipLevel = ras.VipLevelEffective()
}

// dayStart 当日零点时间戳（0 或负数按 0 处理）
func dayStart(ux int64) int64 {
	if ux <= 0 {
		return 0
	}
	t := time.Unix(ux, 0)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Unix()
}
