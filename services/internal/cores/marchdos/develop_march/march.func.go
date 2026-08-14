package develop_march

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"

	"go.uber.org/zap"

	"server.slg.com/common/loggers"
)

// developStep 每次开发提升的等级数（固定值，当前 3，后续可迁配置）
const developStep = cores_declarations.MapLevel(3)

// New 创建开发行军执行器
//
// 开发行军（10005）的生命周期：
//  1. 到达目标 → 校验目标为自己的地且可开发（守军配置存在）
//  2. 战斗     → 与目标等级（当前+3）的守军战斗（复用 battle 节点，PvE）
//  3. 结算     → 胜利：地块升级 +3、标记已开发；失败：召回
//  4. 去留     → 有 IsStay 停留（MarchState_Stay），否则返回（MarchState_Back）
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if !checkDevelopable(mgr, info) {
			// 目标不可开发 → 战败召回（返回 TransitMapID）
			m.DefeatRecall(mgr)
		}
	})

	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return // 战前校验失败已召回
		}

		settle := mgr.GetBattleSettleFunc()
		if settle == nil {
			loggers.Logger.Warn("battle settle func not injected for develop",
				zap.Uint64("march_id", info.GetMarchID().Uint64()))
			m.DefeatRecall(mgr) // 兜底：召回
			return
		}

		req := buildDevelopSettleReq(mgr, info)
		rsp, err := settle(req)
		if err != nil || rsp == nil {
			loggers.Logger.Warn("develop battle settle failed",
				zap.Uint64("march_id", info.GetMarchID().Uint64()),
				zap.Error(err))
			m.DefeatRecall(mgr) // 兜底：召回
			return
		}

		if !rsp.GetAttackerWin() {
			m.DefeatRecall(mgr) // 战斗失败 → 召回
			return
		}

		// 胜利：地块升级 + 标记已开发（只升级不占归属）
		applyDevelopResult(mgr, info)
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		// 战前校验失败/战斗失败已召回（Back），或返回途中 → 不再重复处理
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return
		}
		// 去留：有 IsStay 停留（驻留地块），否则返回（与普通行军一致）
		if info.IsStay {
			info.MarchState = pb_maps_march.MarchState_Stay
			mgr.UpdateMarchPush(info)
			return
		}
		m.CallBack()
	})

	return m
}

// checkDevelopable 校验目标地块可开发：
//   - 目标为自己的地（ownerID == fromRoleID）
//   - 目标地块存在且未达开发上限（守军配置存在）
func checkDevelopable(mgr *map_managers.MapManager, info *marchs.MarchInfo) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok || toMapInfo == nil {
		return false
	}

	// 仅自己的地可开发
	if toMapInfo.GetOwnerID() != info.GetFromRoleID() {
		return false
	}

	// 目标等级（当前+3）必须有守军配置 → 可开发
	guardFunc := mgr.GetGuardConfigFunc()
	if guardFunc == nil {
		loggers.Logger.Warn("guard config func not injected for develop",
			zap.Uint64("march_id", info.GetMarchID().Uint64()))
		return false
	}

	curLevel := int32(toMapInfo.GetLevel())
	targetLevel := curLevel + int32(developStep)
	if slots := guardFunc(cores_declarations.MapLevel(targetLevel)); len(slots) == 0 {
		return false
	}

	return true
}

// buildDevelopSettleReq 组装开发战斗结算请求：
// 攻击方 = 行军队伍；防守方 = 目标等级守军（NPC，role_id=0）。
func buildDevelopSettleReq(mgr *map_managers.MapManager, info *marchs.MarchInfo) *pb_battle.BattleSettleReq {
	req := &pb_battle.BattleSettleReq{
		RoleId:       info.GetFromRoleID(),
		UnionId:      info.GetUnionID(),
		MarchId:      info.GetMarchID().Uint64(),
		MarchType:    int32(info.MarchType),
		MapId:        int32(info.GetToMapID()),
		AttackerTeam: info.GetTeam().Format2Pb(),
	}

	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok || toMapInfo == nil {
		return req
	}

	guardFunc := mgr.GetGuardConfigFunc()
	if guardFunc == nil {
		return req
	}

	curLevel := int32(toMapInfo.GetLevel())
	targetLevel := curLevel + int32(developStep)
	guardSlots := guardFunc(cores_declarations.MapLevel(targetLevel))
	if len(guardSlots) == 0 {
		return req
	}

	req.DefenderGroups = []*pb_battle.DefenderGroup{
		{
			GroupType: pb_battle.DefenderGroupType_DEFENDER_GROUP_STAY_IDLE,
			Marches: []*pb_battle.DefenderMarch{
				{
					RoleId: 0, // NPC 守军
					Team:   &pb_battle.TeamInfo{SlotInfo: guardSlots},
				},
			},
		},
	}

	return req
}

// applyDevelopResult 应用开发胜利结果：地块升级 + 标记已开发（只升级不占归属）。
func applyDevelopResult(mgr *map_managers.MapManager, info *marchs.MarchInfo) {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok || toMapInfo == nil {
		return
	}

	if !toMapInfo.TryLock() {
		return
	}
	defer toMapInfo.UnLock()

	toMapInfo.AddLevel(developStep)
	toMapInfo.MarkDeveloped()

	// 持久化等级/开发状态（当前持锁，Save 只标记脏，SaveDo 稍后刷盘）
	mgr.GetMapDataManager().Save(toMapInfo)

	// 更新视野推送
	mgr.UpdateMapPush(info.GetToMapID())
}
