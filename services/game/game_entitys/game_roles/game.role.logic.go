package game_roles

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
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

// CheckItemEnough 检查道具是否充足
func (r *Role) CheckItemEnough(checkItems []common_declarations.ItemUse) pb_error_code.ErrorCode {
	for _, oneItem := range checkItems {
		switch oneItem.ItemType {
		case pb_confs.ItemTypeNormal:
			return r.GetItems().CheckItemEnough(oneItem)
		}
	}
	return pb_error_code.ErrorCode_ParamError
}

// AddItem 获得道具
func (r *Role) AddItem(addItems []common_declarations.ItemUse, optID, reason string, optTimeUx int64) []*pb_item.ItemChangeLog {
	var vet []*pb_item.ItemChangeLog
	var curCount int64
	for _, oneItem := range addItems {
		switch oneItem.ItemType {
		case pb_confs.ItemTypeNormal:
			curCount = r.GetItems().AddItem(oneItem, optTimeUx)
		}
		oneLog := oneItem.Format2ChangeLogPb(optID, r.ID, oneItem.Count, curCount, optTimeUx, reason)
		vet = append(vet, oneLog)
	}
	return vet
}

// ReduceItem 扣除道具
func (r *Role) ReduceItem(useItems []common_declarations.ItemUse, optID, reason string, optTimeUx int64) []*pb_item.ItemChangeLog {
	var vet []*pb_item.ItemChangeLog
	var curCount int64
	for _, oneItem := range useItems {
		switch oneItem.ItemType {
		case pb_confs.ItemTypeNormal:
			curCount = r.GetItems().ReduceItem(oneItem.ItemID, oneItem.Count)
		}
		oneLog := oneItem.Format2ChangeLogPb(optID, r.ID, -oneItem.Count, curCount, optTimeUx, reason)
		vet = append(vet, oneLog)
	}
	return vet
}
