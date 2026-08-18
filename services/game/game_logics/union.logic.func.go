package game_logics

import (
	"strings"
	"sync"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_union"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_unions"
	"server.slg.com/services/game/game_models"
)

// unionOpLock 联盟写操作全局互斥（demo 级串行，避免并发成员变更竞态）
var unionOpLock sync.Mutex

// roleUnionIndexSync 角色联盟变更 → worldmap 索引同步（由 game 启动层接入 worldmap RPC；默认 no-op）
var roleUnionIndexSync func(roleID, unionID uint64)

// SetRoleUnionIndexSyncFunc 注入联盟索引同步函数（P1-3 接线；game_internals 层调用，避免循环依赖）
func SetRoleUnionIndexSyncFunc(f func(roleID, unionID uint64)) {
	roleUnionIndexSync = f
}

func syncRoleUnionIndex(roleID, unionID uint64) {
	if roleUnionIndexSync != nil {
		roleUnionIndexSync(roleID, unionID)
	}
}

// 常量
const (
	maxUnionNameLen = 24 // 联盟名长度上限（rune）
	maxUnionMembers = 50 // 联盟成员上限
)

// 职位常量
const (
	unionLeaderPos  = int32(pb_union.UnionPosition_UNION_POSITION_LEADER)
	unionOfficerPos = int32(pb_union.UnionPosition_UNION_POSITION_OFFICER)
	unionMemberPos  = int32(pb_union.UnionPosition_UNION_POSITION_MEMBER)
)

// UnionCreate 建盟：角色成为盟主，写入联盟聚合 + 角色侧快照
func UnionCreate(role *game_roles.Role, name string) (*pb_union.RoleUnionInfo, error) {
	name = strings.TrimSpace(name)
	if name == "" || len([]rune(name)) > maxUnionNameLen {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNameInvalid, "invalid union name")
	}
	if role.GetRoleUnion().IsInUnion() {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionAlreadyIn, "already in a union")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	if exists, err := game_unions.GetRepo().ExistsName(name); err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union name check failed")
	} else if exists {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNameExists, "union name exists")
	}

	u := &game_models.Union{
		Name:     name,
		LeaderID: role.ID,
		Members: []*pb_union.UnionMember{{
			RoleId:   role.ID,
			Position: unionLeaderPos,
			RoleName: role.GetAttr().Ensure().RoleName,
			Level:    role.Level(),
		}},
	}
	unionID, err := game_unions.GetRepo().Create(u)
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union create failed")
	}

	role.GetRoleUnion().Set(unionID, name, role.ID, "", unionLeaderPos)
	syncRoleUnionIndex(role.ID, unionID)

	return role.GetRoleUnion().Format2Pb(), nil
}

// UnionJoin 加入联盟（无申请流程，直接加入）
func UnionJoin(role *game_roles.Role, unionID uint64) (*pb_union.RoleUnionInfo, error) {
	if role.GetRoleUnion().IsInUnion() {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionAlreadyIn, "already in a union")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNotFound, "union not found")
	}
	if len(u.Members) >= maxUnionMembers {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionFull, "union full")
	}
	if unionMember(u, role.ID) != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionAlreadyIn, "already member")
	}

	u.Members = append(u.Members, &pb_union.UnionMember{
		RoleId:   role.ID,
		Position: unionMemberPos,
		RoleName: role.GetAttr().Ensure().RoleName,
		Level:    role.Level(),
	})
	if err := game_unions.GetRepo().Save(u); err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union save failed")
	}

	role.GetRoleUnion().Set(unionID, u.Name, u.LeaderID, u.Notice, unionMemberPos)
	syncRoleUnionIndex(role.ID, unionID)

	return role.GetRoleUnion().Format2Pb(), nil
}

// UnionLeave 退出联盟（盟主需先转让/解散）
func UnionLeave(role *game_roles.Role) error {
	if !role.GetRoleUnion().IsInUnion() {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotIn, "not in a union")
	}
	if role.GetRoleUnion().Position() == unionLeaderPos {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionIsLeader, "leader cannot leave, transfer or dissolve first")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	unionID := role.GetRoleUnion().UnionID()
	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u != nil {
		removeUnionMember(u, role.ID)
		if err := game_unions.GetRepo().Save(u); err != nil {
			return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union save failed")
		}
	}

	role.GetRoleUnion().Clear()
	syncRoleUnionIndex(role.ID, 0)
	return nil
}

// UnionKick 踢人：盟主可踢任何人；官员可踢成员
func UnionKick(role *game_roles.Role, targetRoleID uint64) error {
	if !role.GetRoleUnion().IsInUnion() {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotIn, "not in a union")
	}
	pos := role.GetRoleUnion().Position()
	if pos != unionLeaderPos && pos != unionOfficerPos {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNoPermission, "no permission")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	unionID := role.GetRoleUnion().UnionID()
	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotFound, "union not found")
	}
	target := unionMember(u, targetRoleID)
	if target == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotIn, "target not in union")
	}
	if target.GetPosition() == unionLeaderPos {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionIsLeader, "cannot kick leader")
	}

	removeUnionMember(u, targetRoleID)
	if err := game_unions.GetRepo().Save(u); err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union save failed")
	}

	clearRoleUnionSnapshot(targetRoleID)
	syncRoleUnionIndex(targetRoleID, 0)
	return nil
}

// UnionTransfer 转让盟主：原盟主降为成员，目标升为盟主，刷新全盟快照 leader
func UnionTransfer(role *game_roles.Role, targetRoleID uint64) error {
	if role.GetRoleUnion().Position() != unionLeaderPos {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNoPermission, "only leader can transfer")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	unionID := role.GetRoleUnion().UnionID()
	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotFound, "union not found")
	}
	if unionMember(u, targetRoleID) == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNotIn, "target not in union")
	}

	unionMember(u, role.ID).Position = unionMemberPos
	unionMember(u, targetRoleID).Position = unionLeaderPos
	u.LeaderID = targetRoleID
	if err := game_unions.GetRepo().Save(u); err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union save failed")
	}

	// 刷新全盟角色快照（brief leader + 各自职位）
	refreshUnionMemberSnapshots(u)
	return nil
}

// UnionDissolve 解散联盟：盟主，清空全盟成员快照并删除联盟
func UnionDissolve(role *game_roles.Role) error {
	if role.GetRoleUnion().Position() != unionLeaderPos {
		return rpc_results.Error(pb_error_code.ErrorCode_UnionNoPermission, "only leader can dissolve")
	}

	unionOpLock.Lock()
	defer unionOpLock.Unlock()

	unionID := role.GetRoleUnion().UnionID()
	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u != nil {
		for _, m := range u.Members {
			if m.GetRoleId() == role.ID {
				continue
			}
			clearRoleUnionSnapshot(m.GetRoleId())
			syncRoleUnionIndex(m.GetRoleId(), 0)
		}
		if err := game_unions.GetRepo().Delete(unionID); err != nil {
			return rpc_results.Error(pb_error_code.ErrorCode_Failed, "union delete failed")
		}
	}

	role.GetRoleUnion().Clear()
	syncRoleUnionIndex(role.ID, 0)
	return nil
}

// UnionInfo 查询联盟完整信息
func UnionInfo(unionID uint64) (*pb_union.UnionInfo, error) {
	u, err := game_unions.GetRepo().Get(unionID)
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNotFound, "union not found")
	}
	return &pb_union.UnionInfo{
		Brief:   unionBriefPb(u),
		Members: u.Members,
	}, nil
}

// UnionMemberList 当前角色所在联盟的成员列表
func UnionMemberList(role *game_roles.Role) ([]*pb_union.UnionMember, error) {
	if !role.GetRoleUnion().IsInUnion() {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNotIn, "not in a union")
	}
	u, err := game_unions.GetRepo().Get(role.GetRoleUnion().UnionID())
	if err != nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_Failed, "union load failed")
	}
	if u == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_UnionNotFound, "union not found")
	}
	return u.Members, nil
}

// ------------------------------- 内部辅助 -------------------------------

// unionMember 查询成员（无则 nil）
func unionMember(u *game_models.Union, roleID uint64) *pb_union.UnionMember {
	for _, m := range u.Members {
		if m.GetRoleId() == roleID {
			return m
		}
	}
	return nil
}

// removeUnionMember 移除成员；返回是否有变化
func removeUnionMember(u *game_models.Union, roleID uint64) bool {
	for i, m := range u.Members {
		if m.GetRoleId() == roleID {
			u.Members = append(u.Members[:i], u.Members[i+1:]...)
			return true
		}
	}
	return false
}

// unionBriefPb 联盟简要信息
func unionBriefPb(u *game_models.Union) *pb_union.UnionBrief {
	return &pb_union.UnionBrief{
		UnionId:     u.ID,
		Name:        u.Name,
		LeaderId:    u.LeaderID,
		MemberCount: uint32(len(u.Members)),
		Notice:      u.Notice,
	}
}

// refreshUnionMemberSnapshots 按联盟聚合刷新全部成员的角色侧快照（转让盟主/公告变更用）
func refreshUnionMemberSnapshots(u *game_models.Union) {
	for _, m := range u.Members {
		refreshRoleUnionSnapshot(m.GetRoleId(), u, m.GetPosition())
	}
}

// refreshRoleUnionSnapshot 加载角色并刷新联盟快照（用于被踢/转让等他人操作）
func refreshRoleUnionSnapshot(roleID uint64, u *game_models.Union, position int32) {
	if roleID == 0 || u == nil {
		return
	}
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return
	}
	defer poller.Release()
	role.GetRoleUnion().Set(u.ID, u.Name, u.LeaderID, u.Notice, position)
	poller.Save()
}

// clearRoleUnionSnapshot 加载角色并清空联盟快照（被踢/解散）
func clearRoleUnionSnapshot(roleID uint64) {
	if roleID == 0 {
		return
	}
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return
	}
	defer poller.Release()
	if role.GetRoleUnion().IsInUnion() {
		role.GetRoleUnion().Clear()
		poller.Save()
	}
}
