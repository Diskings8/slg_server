package attack_march

// 战斗结算请求组装 — 战斗计算已迁移到 battle 节点（services/battle/battle_logics），
// 本文件保留依赖 MapManager 的防守方构建 + 结算请求组装。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// buildBattleSettleReq 组装战斗结算请求（依赖 MapManager 构建防守方列表）
func buildBattleSettleReq(mgr *map_managers.MapManager, attacker *marchs.MarchInfo,
	toMapID cores_declarations.MapID) *pb_battle.BattleSettleReq {

	req := &pb_battle.BattleSettleReq{
		RoleId:       attacker.GetFromRoleID(),
		MarchId:      attacker.GetMarchID().Uint64(),
		MarchType:    int32(attacker.MarchType),
		AttackerTeam: attacker.GetTeam().Format2Pb(),
		DefenderGroups: []*pb_battle.DefenderGroup{
			{
				GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_ASSIST,
				Marches:   buildDefenderMarchList(buildAssistDefenders(mgr, toMapID)),
			},
			{
				GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_STAY_IDLE,
				Marches:   buildDefenderMarchList(buildStayIdleDefenders(mgr, toMapID, attacker.GetFromRoleID())),
			},
		},
	}

	// 建筑耐久（攻城区分：有建筑→攻城，无建筑→PvE）
	if toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(toMapID); ok && toMapInfo != nil {
		if overlay := toMapInfo.GetOverlayBuilding(); overlay != nil {
			if building := overlay.GetBuilding(); building != nil {
				req.HasBuilding = true
				req.BuildingHp = building.GetBuildingsCurHp()
			}
		}
	}

	return req
}

// buildDefenderMarchList 把防守方行军列表转成 DefenderMarch 快照
func buildDefenderMarchList(infos []*marchs.MarchInfo) []*pb_battle.DefenderMarch {
	if len(infos) == 0 {
		return nil
	}
	list := make([]*pb_battle.DefenderMarch, 0, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		list = append(list, &pb_battle.DefenderMarch{
			MarchId: info.GetMarchID().Uint64(),
			RoleId:  info.GetFromRoleID(),
			Team:    info.GetTeam().Format2Pb(),
		})
	}
	return list
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
