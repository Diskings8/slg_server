package game_logics

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_error_code"
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

		soldierNum := hs.GetSoldierNum()
		teamSlots = append(teamSlots, &pb_battle.TeamSlotInfo{
			SlotId:        int32(i + 1),
			HeroInfo:      hero.Format2Pb(),
			MaxSoldierNum: soldierNum,
			CurAliveNum:   soldierNum,
			CurInjuredNum: 0,
		})
	}

	if len(teamSlots) == 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_ParamError, "no valid hero slot")
	}
	return teamSlots, nil
}
