package game_roles

import (
	"time"
)

// NewTest 测试辅助 — 创建带有基本初始化的角色实例
func NewTest(roleID uint64) *Role {
	r := &Role{
		ID: roleID,
	}
	r.New()
	r.Init()
	return r
}

// ServerID 获取角色所在服务器ID
// 当子模块 attr 存在后，委派给 r.GetAttr().ServerID
func (r *Role) ServerID() uint32 {
	return 0 // TODO 接入 attr 子模块后替换为实际实现
}

// Level 获取角色等级
// 当子模块 builds 存在后，委派给 r.GetBuilds().GetHeadQuarterLevel()
func (r *Role) Level() int32 {
	return 1 // TODO 接入 builds 子模块后替换为实际实现
}

// UnionID 获取联盟ID
// 当子模块 role_union 存在后，委派给 r.GetRoleUnion().UnionID
func (r *Role) UnionID() uint64 {
	return 0 // TODO 接入 role_union 子模块后替换
}

// VIPLevel 获取VIP等级
// 当子模块 attr 存在后，从 attr 中读取 VIP 信息
func (r *Role) VIPLevel() int32 {
	// TODO 接入 attr 子模块后读取实际 VIP 等级
	return 0
}

// IsOnline 是否在线
// 需要接入 gateway 连接管理后实现真实检测
func (r *Role) IsOnline() bool {
	// TODO 通过 gateway stream 检测角色在线状态
	return false
}

// Offline 角色下线
func (r *Role) Offline() {
	_ = time.Now().Unix()
	// TODO 接入 attr 子模块后记录下线时间:
	// r.GetAttr().LogoutAt = time.Now().Unix()
}

// IsCrossServerMap 是否跨服地图迁城
func (r *Role) IsCrossServerMap() bool {
	// TODO 接入 attr 子模块后:
	// return r.GetAttr().MoveMapServerID > 0
	return false
}
