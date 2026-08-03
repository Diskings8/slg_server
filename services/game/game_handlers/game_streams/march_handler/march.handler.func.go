package march_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	pb_confs "server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
)

// HandlerMarchCreate 创建行军 (1000006)
//
// 从已上阵队伍（formation_id）取英雄+士兵构造出征队伍，调用 worldmap 创建行军。
func HandlerMarchCreate(ctx context.Context, roleID uint64, req *pb_maps_march.MarchCreateReq, resp *pb_maps_march.MarchCreateResp) rpc_results.ResultI {
	if req.GetToMapId() < 0 || req.GetMarchType() < 1 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid march params")
	}

	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	// 从已上阵队伍（唯一队列ID）取英雄 + 士兵
	formation := role.GetFormations().GetFormationByID(req.GetFormationId())
	if formation == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "formation not found")
	}

	// 构造出征队伍（英雄快照 + 士兵）
	var teamSlots []*pb_battle.TeamSlotInfo
	for i, hs := range formation.HeroSlots {
		if hs == nil {
			continue // 空槽位跳过
		}
		hero := role.GetHeroes().GetHero(pb_confs.ItemID(hs.GetHeroId()))
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
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "no valid hero slot")
	}

	// 调用 worldmap 创建行军
	createRsp, callErr := game_rpc_clients.WorldMap().CreateMarch(ctx, &pb_worldmap.CreateMarchReq{
		RoleId:    roleID,
		FromMapId: req.GetFromMapId(),
		ToMapId:   req.GetToMapId(),
		MarchType: req.GetMarchType(),
		TeamSlots: teamSlots,
		BaseSpeed: 100,
	})
	if callErr != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("create march failed: %s", callErr.Error()))
	}

	resp.MarchId = createRsp.GetMarchId()
	resp.EndTime = createRsp.GetEndTime()
	return nil
}
