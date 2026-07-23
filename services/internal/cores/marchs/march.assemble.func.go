package marchs

import (
	"slices"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// ---- 集结行军管理（Assemble March） ----

// AssembleCreate 创建集结行军
//
// baseMarchID 是集结发起者的行军 ID，marchInfo 是集结成员的行军信息。
// 如果集结已存在则将成员追加到列表中；不存在则创建新集结。
func (mm *MarchInfoManager) AssembleCreate(baseMarchID cores_declarations.MarchID, marchInfo *MarchInfo) {
	mm.allAssembleMarchLock.Lock()
	defer mm.allAssembleMarchLock.Unlock()

	list, ok := mm.allAssembleMarch[baseMarchID]
	if !ok {
		list = make([]*MarchInfo, 0, 8) // 集结上限通常为 5-8 人
	}
	list = append(list, marchInfo)
	mm.allAssembleMarch[baseMarchID] = list
}

// AssembleJoin 加入集结行军
func (mm *MarchInfoManager) AssembleJoin(baseMarchID cores_declarations.MarchID, marchInfo *MarchInfo) {
	mm.AssembleCreate(baseMarchID, marchInfo)
}

// AssembleLeft 从集结中移除指定成员
//
// 根据 marchID 查找并移除对应的集结成员。
// 如果移除后集结为空，则自动删除该集结组。
// 返回被移除的行军信息。
func (mm *MarchInfoManager) AssembleLeft(marchID cores_declarations.MarchID) *MarchInfo {
	mm.allAssembleMarchLock.Lock()
	defer mm.allAssembleMarchLock.Unlock()

	for baseID, members := range mm.allAssembleMarch {
		for i, m := range members {
			if m.GetMarchID() == marchID {
				members = slices.Delete(members, i, i+1)
				if len(members) == 0 {
					delete(mm.allAssembleMarch, baseID)
				} else {
					mm.allAssembleMarch[baseID] = members
				}
				return m
			}
		}
	}
	return nil
}

// DeleteAssemble 删除整个集结行军组
func (mm *MarchInfoManager) DeleteAssemble(baseMarchID cores_declarations.MarchID) {
	mm.allAssembleMarchLock.Lock()
	defer mm.allAssembleMarchLock.Unlock()
	delete(mm.allAssembleMarch, baseMarchID)
}

// AssembleList 获取集结行军成员列表
func (mm *MarchInfoManager) AssembleList(baseMarchID cores_declarations.MarchID) []*MarchInfo {
	mm.allAssembleMarchLock.RLock()
	defer mm.allAssembleMarchLock.RUnlock()

	list, ok := mm.allAssembleMarch[baseMarchID]
	if !ok {
		return nil
	}
	return list
}

// AssembleRoleIDs 获取集结中所有成员的角色 ID 列表
func (mm *MarchInfoManager) AssembleRoleIDs(baseMarchID cores_declarations.MarchID) []uint64 {
	mm.allAssembleMarchLock.RLock()
	defer mm.allAssembleMarchLock.RUnlock()

	list, ok := mm.allAssembleMarch[baseMarchID]
	if !ok {
		return nil
	}
	ids := make([]uint64, 0, len(list))
	for _, m := range list {
		if m != nil {
			ids = append(ids, m.FromRoleID)
		}
	}
	return ids
}

// AssembleLen 获取集结成员数量
func (mm *MarchInfoManager) AssembleLen(baseMarchID cores_declarations.MarchID) int {
	mm.allAssembleMarchLock.RLock()
	defer mm.allAssembleMarchLock.RUnlock()

	list, ok := mm.allAssembleMarch[baseMarchID]
	if !ok {
		return 0
	}
	return len(list)
}

// RangeAssemble 遍历所有集结组
func (mm *MarchInfoManager) RangeAssemble(f func(baseMarchID cores_declarations.MarchID, members []*MarchInfo) bool) {
	mm.allAssembleMarchLock.RLock()
	defer mm.allAssembleMarchLock.RUnlock()

	for baseID, members := range mm.allAssembleMarch {
		if !f(baseID, members) {
			return
		}
	}
}
