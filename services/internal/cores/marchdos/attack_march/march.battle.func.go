package attack

import (
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// settleBattle 执行战斗结算
//
// 流程：
//  1. 拆迁阶段：攻击方的拆迁值对目标建筑造成伤害（BuildingI.BeAttack）
//  2. 对战阶段：攻击方队伍 vs 防御方驻军
//  3. 计算战损和胜负
func settleBattle(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, toMapID int32) *BattleResult {
	result := &BattleResult{
		Attacker: &BattleSide{
			MarchInfo: attacker,
		},
		Defender: &BattleSide{},
	}

	// ---- 1. 拆迁阶段：对建筑造成伤害 ----
	buildingDamaged := applySiegeDamage(mgr, attacker, toMapID, result)

	// ---- 2. 构建防御方 ----
	defenders := buildDefenders(mgr, toMapID)
	result.Defender.MarchInfo = nil // 防御方可能由多个驻军组成

	if len(defenders) > 0 {
		// ---- 3. 对战阶段 ----
		resolveCombat(attacker, defenders, result)
	} else if !buildingDamaged {
		// 无建筑、无驻军：空地，直接占领
		result.IsOccupied = true
	}

	return result
}

// applySiegeDamage 应用拆迁伤害
// 返回是否造成了建筑伤害
func applySiegeDamage(mgr *map_managers.MapManager, attacker *marchs.MarchInfo, toMapID int32, result *BattleResult) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID)
	if !ok || toMapInfo == nil {
		return false
	}

	overlayBuilding := toMapInfo.GetOverlayBuilding()
	if overlayBuilding == nil {
		return false
	}

	building := overlayBuilding.GetBuilding()
	if building == nil {
		return false
	}

	// 通过 BuildingI.BeAttack 接口造成拆迁伤害
	right, isBroken := building.BeAttack(attacker)
	result.WallDamage = right
	result.IsWallBroken = isBroken
	return right > 0
}

// buildDefenders 构建防御方队伍列表
//
// 防御方来源：
//  1. 目标地块上的驻军（MapAttribute.Assist）
func buildDefenders(mgr *map_managers.MapManager, toMapID int32) []*marchs.MarchInfo {
	toAttribute := mgr.GetMarchManage().MapAttributeGet(toMapID)
	if toAttribute == nil {
		return nil
	}

	defenders := toAttribute.Assist(make([]*marchs.MarchInfo, 0, 8))
	return defenders
}

// resolveCombat 执行攻防对战
//
// 简化模型：比较双方总兵力，按比例结算战损。
func resolveCombat(attacker *marchs.MarchInfo, defenders []*marchs.MarchInfo, result *BattleResult) {
	attTeam := attacker.GetTeam()
	if attTeam == nil {
		result.Attacker.IsDefeated = true
		return
	}

	// 攻击方总战力 = 存活士兵数
	attPower := attTeam.GetAliveSoliderCount()

	// 防御方总战力 = 各驻军存活士兵之和
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

	if attPower == 0 && defPower == 0 {
		// 双方都无兵，攻击方胜（空城）
		result.IsOccupied = true
		return
	}

	if attPower == 0 {
		// 攻击方无兵，直接溃败
		result.Attacker.IsDefeated = true
		return
	}

	if defPower == 0 {
		// 防御方无驻军，攻击方直接获胜
		result.Attacker.AliveSoldiers = attPower
		result.Attacker.WinCountInc = 1
		result.IsOccupied = true
		return
	}

	// 简化战斗：比较兵力
	if attPower >= defPower {
		// 攻击方胜
		// 攻击方战损 = 防御方战力 / 攻击方战力 比例
		attLoss := uint64(float64(attPower) * (float64(defPower) / float64(attPower+defPower)))
		if attLoss > attPower {
			attLoss = attPower
		}
		result.Attacker.AliveSoldiers = attPower - attLoss
		result.Attacker.KilledSoldiers = defPower
		result.Attacker.WinCountInc = 1
		result.IsOccupied = true
		result.IsWallBroken = true // 攻破城墙

		// 防御方全灭
		for _, d := range defenders {
			result.DefenderMarchUpdates = append(result.DefenderMarchUpdates, d)
		}
	} else {
		// 防御方胜
		// 攻击方几乎全灭，只保留少量残兵
		survivors := attPower * 10 / 100 // 保留10%残兵
		if survivors < 1 && attPower > 0 {
			survivors = 1
		}
		result.Attacker.AliveSoldiers = survivors
		result.Attacker.IsDefeated = true
		result.Attacker.KilledSoldiers = defPower / 2 // 击杀一半守军

		// 防御方受伤但未全灭
	}
}

// applyBattleLosses 应用战损到队伍数据
//
// 根据战斗结果更新攻击方和防御方的队伍存活数。
func applyBattleLosses(result *BattleResult) {
	// 攻击方
	if result.Attacker != nil && result.Attacker.MarchInfo != nil {
		attTeam := result.Attacker.MarchInfo.Team
		if attTeam != nil {
			totalAlive := attTeam.GetAliveSoliderCount()
			if totalAlive > 0 && result.Attacker.AliveSoldiers < totalAlive {
				ratio := float64(result.Attacker.AliveSoldiers) / float64(totalAlive)
				for _, slot := range attTeam.Slots {
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
		}
	}

	// 防御方驻军
	for _, defender := range result.DefenderMarchUpdates {
		if defender == nil {
			continue
		}
		defTeam := defender.Team
		if defTeam == nil {
			continue
		}
		totalDef := defTeam.GetAliveSoliderCount()
		if totalDef == 0 {
			continue
		}
		// 防御方驻军全灭
		for _, slot := range defTeam.Slots {
			if slot == nil {
				continue
			}
			slot.CurAliveNum = 0
		}
	}
}
