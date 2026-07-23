package map_datas

import (
	"maps"
	"sync"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// UnionMemberMapIDs 联盟成员地图位置索引
//
// 维护三个维度的映射关系：
//   - unionRoleMapID: 联盟 ID → 角色 ID → 地图格子 ID
//   - roleUnionID:    角色 ID → 联盟 ID
//   - roleMap:        角色 ID → 地图格子 ID
//
// 提供 O(1) 的角色归属查询和联盟成员遍历能力。
// 适用于联盟战争、联盟领地展示、联盟成员视野等场景。
type UnionMemberMapIDs struct {
	unionRoleMapID map[uint64]map[uint64]cores_declarations.MapID
	roleUnionID    map[uint64]uint64
	roleMap        map[uint64]cores_declarations.MapID
	locker         sync.RWMutex
}

// NewUnionMemberMapIDs 创建联盟成员地图位置索引
func NewUnionMemberMapIDs() *UnionMemberMapIDs {
	return &UnionMemberMapIDs{
		unionRoleMapID: make(map[uint64]map[uint64]cores_declarations.MapID),
		roleUnionID:    make(map[uint64]uint64),
		roleMap:        make(map[uint64]cores_declarations.MapID),
	}
}

// Set 设置角色所属联盟和地图位置
//
// 如果角色之前属于其他联盟，会自动从旧联盟中移除。
func (u *UnionMemberMapIDs) Set(unionID, roleID uint64, mapID cores_declarations.MapID) {
	u.locker.Lock()
	defer u.locker.Unlock()

	if unionID > 0 {
		if _, ok := u.unionRoleMapID[unionID]; !ok {
			u.unionRoleMapID[unionID] = make(map[uint64]cores_declarations.MapID)
		}
		u.unionRoleMapID[unionID][roleID] = mapID
	}

	if oldUnionID := u.roleUnionID[roleID]; oldUnionID != unionID {
		delete(u.unionRoleMapID[oldUnionID], roleID)
	}

	u.roleUnionID[roleID] = unionID
	u.roleMap[roleID] = mapID
}

// SetUnionID 更新角色所属联盟
//
// 角色已在地图上时，仅变更联盟归属关系，位置不变。
func (u *UnionMemberMapIDs) SetUnionID(roleID, unionID uint64) {
	u.locker.Lock()
	defer u.locker.Unlock()

	if oldUnionID := u.roleUnionID[roleID]; oldUnionID != unionID {
		delete(u.unionRoleMapID[oldUnionID], roleID)
	}

	mapID := u.roleMap[roleID]
	if _, ok := u.unionRoleMapID[unionID]; !ok {
		u.unionRoleMapID[unionID] = make(map[uint64]cores_declarations.MapID)
	}
	u.unionRoleMapID[unionID][roleID] = mapID
	u.roleUnionID[roleID] = unionID
}

// SetMapID 更新角色在地图上的位置
func (u *UnionMemberMapIDs) SetMapID(roleID uint64, mapID cores_declarations.MapID) {
	u.locker.Lock()
	defer u.locker.Unlock()

	if unionID, ok := u.roleUnionID[roleID]; ok && unionID > 0 {
		u.unionRoleMapID[unionID][roleID] = mapID
	}
	u.roleMap[roleID] = mapID
}

// Remove 移除角色
//
// 角色离开地图（退出游戏/迁城/删除角色）时调用。
func (u *UnionMemberMapIDs) Remove(roleID uint64) {
	u.locker.Lock()
	defer u.locker.Unlock()

	oldUnionID := u.roleUnionID[roleID]
	if _, ok := u.unionRoleMapID[oldUnionID]; ok {
		delete(u.unionRoleMapID[oldUnionID], roleID)
	}
	delete(u.roleUnionID, roleID)
	delete(u.roleMap, roleID)
}

// GetUnionRoleMapIDs 获取联盟所有成员的地图位置
//
// 返回 map[角色ID]地图格子ID 的副本，调用方可安全修改。
func (u *UnionMemberMapIDs) GetUnionRoleMapIDs(unionID uint64) map[uint64]cores_declarations.MapID {
	u.locker.RLock()
	defer u.locker.RUnlock()

	data, ok := u.unionRoleMapID[unionID]
	if !ok {
		return nil
	}
	return maps.Clone(data)
}

// GetUnionRoleIDs 获取联盟在地图上的所有角色 ID
func (u *UnionMemberMapIDs) GetUnionRoleIDs(unionID uint64) []uint64 {
	u.locker.RLock()
	defer u.locker.RUnlock()

	if _, ok := u.unionRoleMapID[unionID]; !ok {
		return nil
	}
	out := make([]uint64, 0, len(u.unionRoleMapID[unionID]))
	for roleID := range u.unionRoleMapID[unionID] {
		out = append(out, roleID)
	}
	return out
}

// GetRoleUnionID 获取角色所属联盟 ID
func (u *UnionMemberMapIDs) GetRoleUnionID(roleID uint64) uint64 {
	u.locker.RLock()
	defer u.locker.RUnlock()
	return u.roleUnionID[roleID]
}

// GetRoleMapID 获取角色所在的地图格子 ID
func (u *UnionMemberMapIDs) GetRoleMapID(roleID uint64) (cores_declarations.MapID, bool) {
	u.locker.RLock()
	defer u.locker.RUnlock()
	mapInfo, ok := u.roleMap[roleID]
	return mapInfo, ok
}

// Len 返回索引中的总角色数
func (u *UnionMemberMapIDs) Len() int {
	u.locker.RLock()
	defer u.locker.RUnlock()
	return len(u.roleMap)
}
