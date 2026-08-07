package game_logics

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// MarchBuildTeam 根据请求编队构造出征队伍（英雄快照 + 士兵）
//
// 前置：编队存在；空槽位跳过；无有效英雄报错。
// 返回: teamSlots（供 worldmap CreateMarch 使用）。
func MarchBuildTeam(role *game_roles.Role, req *pb_maps_march.MarchCreateReq) ([]*pb_battle.TeamSlotInfo, rpc_results.ResultI) {
	// 从已上阵队伍（唯一队列ID）取英雄 + 士兵
	formation := role.GetFormations().GetFormationByID(req.GetFormationId())
	if formation == nil {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation not found")
	}

	teamSlots := make([]*pb_battle.TeamSlotInfo, 0)
	for i, hs := range formation.HeroSlots {
		if hs == nil {
			continue // 空槽位跳过
		}
		hero := role.GetHeroes().GetHero(hs.GetHeroId())
		if hero == nil {
			continue
		}

		// 出征兵力：当前携带兵力按英雄等级+兵营上限封顶（formation 存战损后存活值）
		maxSoldier := soldierLimitOf(role, formation.CityID, hero.GetLevel())
		curSoldier := hs.GetSoldierNum()
		if maxSoldier > 0 && curSoldier > maxSoldier {
			curSoldier = maxSoldier
		}
		// 英雄信息承载出征战斗状态（实例ID/兵力/攻击距离），TeamSlotInfo 只留槽位
		heroInfo := hero.Format2Pb()
		heroInfo.HeroId = hs.GetHeroId()
		heroInfo.AttackRange = attackRangeOf(hero.GetHeroConfID())
		heroInfo.SoldierInfo = &pb_hero.SoldierInfo{
			MaxSoldierNum: maxSoldier,
			CurAliveNum:   curSoldier,
			CurInjuredNum: 0,
		}

		teamSlots = append(teamSlots, &pb_battle.TeamSlotInfo{
			SlotId:   int32(i), // 0 基：0=大营，1=1号位，2=2号位
			HeroInfo: heroInfo,
		})
	}

	if len(teamSlots) == 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_ParamError, "no valid hero slot")
	}
	return teamSlots, nil
}

// attackRangeOf 英雄攻击距离（读 hero 配置；未配置按 0 = 无法攻击到目标，由配置保证填写）
func attackRangeOf(confID int32) uint32 {
	if hc, ok := game_conf.Load().Hero.HeroConf(confID); ok {
		return hc.AttackRange
	}
	return 0
}
