package attack_march

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

// New 创建攻击行军执行器
//
// 攻击行军（10001）到达流水线：
//
//	Prepare → 战前合法性校验 → 不合格则直接返回
//	Do      → 组装结算请求 → 调 battle 节点 RPC 结算 → 应用结果（伤亡/占城/状态/耐久）
//	Finish  → 战报推送 → 事件触发
func New(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	m := marchdos.NewSingleMarch(mm)
	m.SetMarchInfo(marchInfo)

	if fromInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetFromMapID()); ok {
		m.SetFromMapInfo(fromInfo)
	}
	if toInfo, ok := mm.GetMapDataManager().GetMapInfo(marchInfo.GetToMapID()); ok {
		m.SetToMapInfo(toInfo)
	}

	var battleRsp *pb_battle.BattleSettleRsp

	// ---- Prepare：战前校验 ----
	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if !checkTargetLegality(mgr, info) {
			// 目标不合法 → 战败召回（返回 TransitMapID）
			m.DefeatRecall(mgr)
		}
	})

	// ---- Do：组装请求 → 调 battle 节点结算 → 应用结果 ----
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
			loggers.Logger.Warn("battle settle func not injected",
				zap.Uint64("march_id", info.GetMarchID().Uint64()))
			m.DefeatRecall(mgr) // 兜底：战败召回（返回 TransitMapID）
			return
		}

		req := buildBattleSettleReq(mgr, info, info.GetToMapID())
		rsp, err := settle(req)
		if err != nil || rsp == nil {
			loggers.Logger.Warn("battle settle failed",
				zap.Uint64("march_id", info.GetMarchID().Uint64()),
				zap.Error(err))
			m.DefeatRecall(mgr) // 兜底：战败召回（返回 TransitMapID）
			return
		}

		battleRsp = rsp
		applyBattleSettleRsp(mgr, info, rsp)

		// 战败 → 优先返回当前的 TransitMapID
		if !rsp.GetAttackerWin() {
			m.DefeatRecall(mgr)
		}
	})

	// ---- Finish：战报推送 + 事件触发 ----
	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil || battleRsp == nil {
			return
		}

		// 战果挂到行军信息，到达事件发布时随 MarchEvent 回传 game（发放英雄经验）
		info.BattleResult = buildMarchBattleResult(battleRsp)

		pushBattleResult(mgr, info, battleRsp.GetAttackerWin())
		triggerBattleEvents(mgr, info)
	})

	return m
}

// checkTargetLegality 战前目标合法性校验
func checkTargetLegality(mgr *map_managers.MapManager, info *marchs.MarchInfo) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok {
		return false
	}

	if toMapInfo.GetOwnerID() == 0 && toMapInfo.GetOverlayBuilding() == nil {
		return false
	}

	// 目标处于保护期内（地块刚被释放，战乱地外）→ 不可攻击
	if toMapInfo.GetProtectedEndTime() > time.Now().Unix() {
		return false
	}

	// TODO: 校验是否是盟友（需接入联盟数据）

	return true
}
