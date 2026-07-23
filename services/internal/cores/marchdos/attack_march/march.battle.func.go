package attack_march

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// settleBattle 执行战斗结算
//
// 流程：
//  1. PvP：逐层战斗，每层产出 OneBattleResult
//     驻守(assist) → 停留(stay) → Idle
//  2. PvE/攻城：
//     有建筑 → 攻城（拆迁 vs 建筑耐久）
//     无建筑 → PvE（拆迁 vs 地块耐久）
func settleBattle(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, toMapID cores_declarations.MapID) *BattleResult {
	result := &BattleResult{
		Layers:            make([]*LayerResult, 0, 3),
		FinalAttackerInfo: attacker,
	}

	// ---- 1. PvP 阶段：逐层战斗 ----
	if !resolveDefendersLayer(mgr, attacker, toMapID, result) {
		return result
	}

	// ---- 2. 建筑/地块 阶段 ----
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return result
	}

	if overlayBuilding := toMapInfo.GetOverlayBuilding(); overlayBuilding != nil {
		if building := overlayBuilding.GetBuilding(); building != nil {
			// 有建筑 → 攻城
			_, isBroken := building.BeAttack(attacker)
			layer := &LayerResult{
				Attacker:   buildSideFromTeam(attacker),
				Defender:   &pb_battle.BattleSide{},
				IsOccupied: isBroken,
			}
			result.Layers = append(result.Layers, layer)
			return result
		}
	}

	// 无建筑 → PvE
	resolvePvE(mgr, attacker, toMapInfo, result)
	return result
}

// resolvePvE 无建筑时的 PvE 战斗
func resolvePvE(_ *map_managers.MapManager, _ *marchs.MarchInfo, _ *map_datas.MapInfo, result *BattleResult) {
	layer := &LayerResult{
		Attacker:   buildSideFromTeam(result.FinalAttackerInfo),
		Defender:   &pb_battle.BattleSide{},
		IsOccupied: true,
	}
	result.Layers = append(result.Layers, layer)
}

// resolveDefendersLayer 逐层攻克防守方
//
// 顺序：assist(驻守) → stay(停留) → idle
// 每层产出 LayerResult 追加到 result.Layers
func resolveDefendersLayer(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, toMapID cores_declarations.MapID, result *BattleResult) bool {
	assistDefenders := buildAssistDefenders(mgr, toMapID)
	if len(assistDefenders) > 0 {
		if !fightLayer(attacker, assistDefenders, result) {
			return false
		}
	}

	stayIdleDefenders := buildStayIdleDefenders(mgr, toMapID, attacker.GetFromRoleID())
	if len(stayIdleDefenders) > 0 {
		if !fightLayer(attacker, stayIdleDefenders, result) {
			return false
		}
	}

	return true
}

// fightLayer 攻击方与某一层防守方交战
//
// 产出 LayerResult 追加到 result.Layers。
// 返回 false 表示攻击方溃败。
func fightLayer(attacker *marchs.MarchInfo, defenders []*marchs.MarchInfo, result *BattleResult) bool {
	attTeam := attacker.GetTeam()
	if attTeam == nil {
		return false
	}

	attPower := attTeam.GetAliveSoliderCount()
	if attPower == 0 {
		return false
	}

	var defPower uint64
	for _, d := range defenders {
		if d == nil {
			continue
		}
		team := d.GetTeam()
		if team == nil {
			continue
		}
		defPower += team.GetAliveSoliderCount()
	}

	if defPower == 0 {
		return true
	}

	layer := &LayerResult{
		DefeatedMarches: defenders,
	}

	aliveBefore := attTeam.GetAliveSoliderCount()

	if attPower >= defPower {
		// 攻击方胜
		loss := uint64(float64(attPower) * (float64(defPower) / float64(attPower+defPower)))
		if loss > attPower {
			loss = attPower
		}
		remain := attPower - loss

		layer.Attacker = &pb_battle.BattleSide{
			KilledSoldiers: defPower,
			TeamInfo:       cloneTeamInfo(attTeam),
		}
		layer.Defender = &pb_battle.BattleSide{
			IsDefeated:     true,
			KilledSoldiers: 0,
		}

		applyLossesToTeam(attTeam, attPower, remain, aliveBefore)
		result.Layers = append(result.Layers, layer)
		return true
	}

	// 攻击方败
	survivors := attPower * 10 / 100
	if survivors < 1 {
		survivors = 1
	}

	layer.Attacker = &pb_battle.BattleSide{
		IsDefeated:     true,
		KilledSoldiers: defPower / 2,
		TeamInfo:       cloneTeamInfo(attTeam),
	}
	layer.Defender = &pb_battle.BattleSide{
		KilledSoldiers: 0,
	}

	applyLossesToTeam(attTeam, attPower, survivors, aliveBefore)
	result.Layers = append(result.Layers, layer)
	result.WinCountInc = 0
	return false
}

// buildSideFromTeam 从队伍快照构建 BattleSide
func buildSideFromTeam(info *marchs.MarchInfo) *pb_battle.BattleSide {
	team := info.GetTeam()
	if team == nil {
		return &pb_battle.BattleSide{}
	}
	return &pb_battle.BattleSide{
		TeamInfo: team.Format2Pb(),
	}
}

// cloneTeamInfo 克隆当前队伍信息为 PB 快照
func cloneTeamInfo(team *marchs.Team) *pb_battle.TeamInfo {
	if team == nil {
		return nil
	}
	return team.Format2Pb()
}

// applyLossesToTeam 按比例减少队伍各 slot 存活数
func applyLossesToTeam(team *marchs.Team, beforePower, afterPower, totalAliveBefore uint64) {
	if team == nil || beforePower == 0 {
		return
	}
	ratio := float64(afterPower) / float64(beforePower)
	for _, slot := range team.Slots {
		if slot == nil {
			continue
		}
		if slot.GetHeroInfo().GetCurStatus() == pb_hero.Status_Injured {
			continue
		}
		alive := uint32(float64(slot.GetCurAliveNum()) * ratio)
		if alive < 1 && slot.GetCurAliveNum() > 0 {
			alive = 1
		}
		slot.CurAliveNum = alive
	}
}

// buildAssistDefenders 构建驻军防守方列表（MarchState_Station）
func buildAssistDefenders(mgr *map_managers.MapManager, toMapID cores_declarations.MapID) []*marchs.MarchInfo {
	attr := mgr.GetMarchManage().MapAttributeGet(toMapID)
	if attr == nil {
		return nil
	}
	return attr.Assist(make([]*marchs.MarchInfo, 0, 8))
}

// buildStayIdleDefenders 构建停留/等待防守方列表
func buildStayIdleDefenders(mgr *map_managers.MapManager, toMapID cores_declarations.MapID, attackerRoleID uint64) []*marchs.MarchInfo {
	attr := mgr.GetMarchManage().MapAttributeGet(toMapID)
	if attr == nil {
		return nil
	}

	var defenders []*marchs.MarchInfo
	attr.RangeMapMarch(func(_ cores_declarations.MarchID, info *marchs.MarchInfo) bool {
		if info == nil {
			return true
		}
		if info.GetFromRoleID() == attackerRoleID {
			return true
		}
		state := info.GetMarchState()
		if state == pb_maps_march.MarchState_Stay || state == pb_maps_march.MarchState_Idle {
			defenders = append(defenders, info)
		}
		return true
	})
	return defenders
}
