package marchs

import (
	"slices"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// Assist 返回援军
// 参数需要为外部传入的 []*MarchInfo{} 这样可以减少大量的分配到堆里需回收的对象
func (ma *MapAttribute) Assist(assistSlice []*MarchInfo) []*MarchInfo {
	if ma == nil {
		return assistSlice
	}

	ma.assistLocker.RLock()
	defer ma.assistLocker.RUnlock()
	assistSlice = slices.Clone(ma.assistSlice)
	return assistSlice
}

// AssistRoleMap 返回援军
func (ma *MapAttribute) AssistRoleMap(out map[uint64]*MarchInfo) map[uint64]*MarchInfo {
	ma.assistLocker.RLock()
	defer ma.assistLocker.RUnlock()
	for _, v := range ma.assistSlice {
		out[v.GetFromRoleID()] = v
	}
	return out
}

// AssistLen 援军数量
func (ma *MapAttribute) AssistLen() (l int) {
	if ma == nil {
		return l
	}

	ma.assistLocker.RLock()
	defer ma.assistLocker.RUnlock()
	return len(ma.assistSlice)
}

// AssistRoleID 援军角色ID
func (ma *MapAttribute) AssistRoleID() (roleIDs []uint64) {
	if ma == nil {
		return roleIDs
	}
	ma.assistLocker.RLock()
	defer ma.assistLocker.RUnlock()
	for _, v := range ma.assistSlice {
		roleIDs = append(roleIDs, v.GetFromRoleID())
	}
	return roleIDs
}

// AssistArrive 援军到达
func (ma *MapAttribute) AssistArrive(m *MarchInfo) {
	if ma == nil {
		return
	}

	ma.assistLocker.Lock()
	defer ma.assistLocker.Unlock()
	if !slices.ContainsFunc(ma.assistSlice, func(tmp *MarchInfo) bool {
		return tmp.GetMarchID() == m.GetMarchID()
	}) {
		ma.assistSlice = append(ma.assistSlice, m)
	}
}

// AssistCallBack 援军返回
func (ma *MapAttribute) AssistCallBack(marchID cores_declarations.MarchID) {
	if ma == nil {
		return
	}
	ma.assistLocker.Lock()
	defer ma.assistLocker.Unlock()
	ma.assistSlice = slices.DeleteFunc(ma.assistSlice, func(tmp *MarchInfo) bool { return tmp.GetMarchID() == marchID })
}

// RangeAssist 返回援军
func (ma *MapAttribute) RangeAssist(f func(marchInfo *MarchInfo) bool) {
	if ma == nil {
		return
	}
	tmp := ma.Assist(make([]*MarchInfo, 0, ma.AssistLen()))
	for _, v := range tmp {
		if !f(v) {
			return
		}
	}
}
