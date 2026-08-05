package battle_logics

// 战斗经验结算 — 每场（攻方 vs 单个守方行军）经验 = 敌方平均等级 × 击杀敌兵 × 系数 ÷ 参战英雄。
// 经验记录在 OneBattleResult.HeroExp（随战报 Results 落库）；game 回调后 HeroAddExp 落地（判升级/属性点）。
// 击杀口径 = 死亡兵数（不含伤兵）。升级由 game 侧结算（战斗内不升级、不影响属性）。

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"
)

// grantBattleExp 本场战斗经验结算。
//
//   - killedEnemy = 本场被击杀的敌兵（死亡兵数）
//   - defSlots    = 本场敌方队伍（计算平均等级）
//   - 参战英雄    = 存活（有有效兵力）英雄，经验平均分配（整数除法）
func grantBattleExp(attackerSlots []*pb_battle.TeamSlotInfo, defSlots []*pb_battle.TeamSlotInfo,
	killedEnemy uint64, result *pb_battle.OneBattleResult) {

	if result == nil || killedEnemy == 0 {
		return
	}
	bc := game_conf.Load().Battle
	if bc == nil || bc.BattleExpCoeff == 0 {
		return
	}
	enemyAvg := avgLevelSlots(defSlots)
	if enemyAvg == 0 {
		return
	}
	total := uint64(enemyAvg) * killedEnemy * uint64(bc.BattleExpCoeff)

	// 参战英雄 = 存活（未受伤且有有效兵力）
	var participants []*pb_battle.TeamSlotInfo
	for _, s := range attackerSlots {
		if s == nil || s.GetHeroInfo() == nil {
			continue
		}
		if s.GetHeroInfo().GetCurStatus() == pb_hero.Status_Injured || s.GetCurAliveNum() == 0 {
			continue
		}
		participants = append(participants, s)
	}
	if len(participants) == 0 {
		return
	}
	perHero := uint32(total / uint64(len(participants)))
	if perHero == 0 {
		return
	}

	for _, s := range participants {
		result.HeroExp = append(result.HeroExp, &pb_battle.HeroExpGain{
			HeroId: s.GetHeroId(),
			Exp:    perHero,
		})
	}
}

// avgLevelSlots 队伍平均等级（所有英雄 cur_level 均值）
func avgLevelSlots(slots []*pb_battle.TeamSlotInfo) uint32 {
	var sum, count uint64
	for _, s := range slots {
		if s == nil || s.GetHeroInfo() == nil {
			continue
		}
		sum += uint64(s.GetHeroInfo().GetCurLevel())
		count++
	}
	if count == 0 {
		return 0
	}
	return uint32(sum / count)
}
