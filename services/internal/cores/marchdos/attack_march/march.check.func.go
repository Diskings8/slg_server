package attack

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// checkTargetLegality 战前校验目标地块合法性
//
// 检查项：
//   - 目标地块存在
//   - 目标不是攻击方自己的地块
//   - 保护期检查
//   - 建筑前置检查（BuildingI.BeforeBeAttack）
func checkTargetLegality(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, toMapID cores_declarations.MapID) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return false
	}

	// 不能攻击自己的地块
	if toMapInfo.GetOwnerID() == attacker.GetFromRoleID() {
		return false
	}

	// 检查叠加建筑的前置条件
	overlayBuilding := toMapInfo.GetOverlayBuilding()
	if overlayBuilding != nil {
		building := overlayBuilding.GetBuilding()
		if building != nil {
			if !building.BeforeBeAttack(attacker) {
				return false
			}
		}
	}

	return true
}
