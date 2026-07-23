package map_managers

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/marchs"
)

const warVisionScanRadius int32 = 2 // 战争视野扫描半径（5×5 范围）

// PushWarInformation 推送战争情报
//
// 攻击方（行军发起者）：收到行军状态更新
// 防守方（建筑战争视野）：扫描目标地块周围 5×5 范围的建筑，
// 判断建筑视野是否覆盖目标，覆盖则推送地块更新（战争视野范围内的玩家会收到）。
func (mm *MapManager) PushWarInformation(marchInfo *marchs.MarchInfo) {
	if marchInfo == nil {
		return
	}
	mm.UpdateMarchPush(marchInfo)
	mm.pushWarAlert(marchInfo.GetToMapID())
}

// PushBattleResult 推送战斗结果
func (mm *MapManager) PushBattleResult(marchInfo *marchs.MarchInfo, attackerWin bool) {
	if marchInfo == nil {
		return
	}
	mm.UpdateMarchPush(marchInfo)
	if attackerWin {
		mm.pushWarAlert(marchInfo.GetToMapID())
	}
}

// pushWarAlert 战争视野推送
//
// 通过 AOI 九宫格推送目标地块更新，覆盖所有在战争视野范围内的在线角色。
// AOI 的范围 + 地块归属者的 AOI 连接已包含建筑归属者和附近玩家。
func (mm *MapManager) pushWarAlert(targetMapID cores_declarations.MapID) {
	// 扫描战争视野，额外推送关键地块
	extraMapIDs := mm.scanWarVision(targetMapID)
	for _, mapID := range extraMapIDs {
		mm.UpdateMapPush(mapID)
	}
	// 目标地块推送（自动覆盖 AOI 九宫格范围的在线玩家）
	mm.UpdateMapPush(targetMapID)
}

// scanWarVision 扫描战争视野范围内的建筑，返回需要额外推送的地块 ID
//
// 算法：
//  1. 以目标地块为中心 5×5 范围扫描
//  2. 有建筑 → 判断建筑视野是否覆盖目标
//  3. 视野覆盖 → 也推送该建筑所在地块，使建筑归属者收到预警
func (mm *MapManager) scanWarVision(targetMapID cores_declarations.MapID) []cores_declarations.MapID {
	var extraMapIDs []cores_declarations.MapID
	mdm := mm.GetMapDataManager()
	config := mdm.GetConfig()

	targetX, targetY := config.MapID2XY(targetMapID)

	for dx := -warVisionScanRadius; dx <= warVisionScanRadius; dx++ {
		for dy := -warVisionScanRadius; dy <= warVisionScanRadius; dy++ {
			scanMapID := config.XY2MapID(targetX+dx, targetY+dy)
			if scanMapID < 0 {
				continue
			}

			info, ok := mdm.GetMapInfo(scanMapID)
			if !ok {
				continue
			}

			overlay := info.GetOverlayBuilding()
			if overlay == nil {
				continue
			}
			building := overlay.GetBuilding()
			if building == nil {
				continue
			}

			visionRange := building.VisionRange()
			if visionRange <= 0 {
				continue
			}

			buildingX, buildingY := config.MapID2XY(scanMapID)
			dist := abs32(targetX-buildingX) + abs32(targetY-buildingY)
			if dist <= visionRange {
				extraMapIDs = append(extraMapIDs, scanMapID)
			}
		}
	}

	// 目标地块自身总是需要推送
	extraMapIDs = append(extraMapIDs, targetMapID)
	return extraMapIDs
}

func abs32(x int32) int32 {
	if x < 0 {
		return -x
	}
	return x
}
