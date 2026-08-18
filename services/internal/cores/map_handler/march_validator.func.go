package map_handler

import (
	"errors"
	"time"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// CreateMarchCtx 创建行军的校验结果
type CreateMarchCtx struct {
	MarchInfo *marchs.MarchInfo
	FromMapID cores_declarations.MapID
	ToMapID   cores_declarations.MapID
	RoleID    uint64
	UnionID   uint64
}

// ValidateCreateMarch 校验创建行军请求
//
// 校验项：
//   - 来源和目标地块存在
//   - 队伍非空且可战斗
//   - 目标地块合法性（有归属或可攻击）
//   - TODO: 体力检查
func ValidateCreateMarch(mm *map_managers.MapManager, info *marchs.MarchInfo) (*CreateMarchCtx, error) {
	if info == nil {
		return nil, errors.New("行军信息为空")
	}
	if info.GetTeam() == nil || !info.GetTeam().CheckCanFight() {
		return nil, errors.New("队伍不可战斗")
	}

	fromInfo, fromOk := mm.GetMapDataManager().GetMapInfo(info.GetFromMapID())
	toInfo, toOk := mm.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !fromOk || !toOk {
		return nil, errors.New("来源或目标地块不存在")
	}

	_ = fromInfo

	// 按行军类型分流目标合法性校验
	switch info.MarchType {
	case cores_declarations.MarchTypeAttack:
		// 攻击：目标有归属或有建筑（可攻占）；或中立资源地（打守军 PvE 占领空地）
		if toInfo.GetOwnerID() == 0 && toInfo.GetOverlayBuilding() == nil {
			if !toInfo.GetElementType().IsResource() {
				return nil, errors.New("目标地块无效")
			}
		}
		// 目标处于保护期内（地块刚被释放）→ 不可攻击
		if toInfo.GetProtectedEndTime() > time.Now().Unix() {
			return nil, errors.New("目标地块保护期内")
		}
	case cores_declarations.MarchTypeDevelop:
		// 开发：仅自己的地可开发
		if toInfo.GetOwnerID() != info.GetFromRoleID() {
			return nil, errors.New("只能开发自己的地块")
		}
	default:
		// 其他类型（assist/sweep/strategy 等）暂不在此强校验
	}

	// TODO: 检查体力

	return &CreateMarchCtx{
		MarchInfo: info,
		FromMapID: info.GetFromMapID(),
		ToMapID:   info.GetToMapID(),
		RoleID:    info.GetFromRoleID(),
		UnionID:   info.GetUnionID(),
	}, nil
}
