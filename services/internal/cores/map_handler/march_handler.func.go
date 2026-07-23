package map_handler

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/marchdos/march_factory"
	"server.slg.com/services/internal/cores/marchs"
)

// CreateMarch 创建行军
//
// 流程：校验 → 持久化 → 挂载 MapAttribute → AOI 设置 → 推送
// 行军出发时目标已定，不可中途变更。
func (h *MarchHandler) CreateMarch(info *marchs.MarchInfo) error {
	mm := h.Manage()
	if mm == nil {
		return cores_declarations.ErrManagerNil
	}

	ctx, err := ValidateCreateMarch(mm, info)
	if err != nil {
		return err
	}

	marchMgr := mm.GetMarchManage()

	if err := marchMgr.CreateMarch(ctx.MarchInfo); err != nil {
		return err
	}

	mm.MarchAOISetupSingle(ctx.MarchInfo)
	mm.UpdateMarchPush(ctx.MarchInfo)
	mm.UpdateMapPush(ctx.FromMapID, ctx.ToMapID)
	return nil
}

// CallBack 召回行军
func (h *MarchHandler) CallBack(marchInfo *marchs.MarchInfo) error {
	mm := h.Manage()
	if mm == nil {
		return cores_declarations.ErrManagerNil
	}
	handle := march_factory.NewMarchDo(mm, marchInfo)
	if handle == nil {
		return cores_declarations.ErrUnknownMarchType
	}
	return handle.CallBack()
}

// CallBackNow 立即召回行军
func (h *MarchHandler) CallBackNow(marchInfo *marchs.MarchInfo) error {
	mm := h.Manage()
	if mm == nil {
		return cores_declarations.ErrManagerNil
	}
	handle := march_factory.NewMarchDo(mm, marchInfo)
	if handle == nil {
		return cores_declarations.ErrUnknownMarchType
	}
	return handle.CallBackNow()
}

// MarchInfo 查询行军信息
func (h *MarchHandler) MarchInfo(marchID cores_declarations.MarchID) *marchs.MarchInfo {
	if h.March == nil {
		return nil
	}
	return h.March().GetMarchInfo(marchID)
}

// MyMarch 查询角色的所有行军
func (h *MarchHandler) MyMarch(roleID uint64) []*marchs.MarchInfo {
	if h.March == nil {
		return nil
	}
	var result []*marchs.MarchInfo
	for _, info := range h.March().AllMarch() {
		if info.GetFromRoleID() == roleID {
			result = append(result, info)
		}
	}
	return result
}
