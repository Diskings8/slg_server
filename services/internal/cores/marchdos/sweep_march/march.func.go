package sweep_march

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_events"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

// EventHandler 处理扫荡事件分支（不打守军，按事件类型执行对应逻辑）。
// 事件类型由"审查"等玩法定义并通过 RegisterEventHandler 注册。
type EventHandler func(mgr *map_managers.MapManager, marchInfo *marchs.MarchInfo, event *map_events.OverlayEvent) error

// eventHandlers 事件类型 → 处理器注册表
var eventHandlers = map[map_events.EventType]EventHandler{}

// RegisterEventHandler 注册事件类型处理器
func RegisterEventHandler(t map_events.EventType, h EventHandler) {
	eventHandlers[t] = h
}

// New 创建扫荡行军执行器
//
// 扫荡行军（10003）生命周期：
//  1. 到达目标，校验目标事件：
//     - 有 overlayEvent 且事件ID == MarchInfo.TargetEventID（出发前记录）且未过期 → 事件分支（不打守军）
//     - 否则 → 与目标当前等级守军 PvE
//  2. 结算完成 → 返回
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
		// 事件分支：目标仍有匹配事件 → 不打守军
		if matchingEvent(m) != nil {
			return
		}
		// 无事件/事件不一致/已过期 → 守军PvE：目标需有守军配置，否则战败召回
		if !checkGuardTarget(mgr, info) {
			m.DefeatRecall(mgr)
			return
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

		// ---- 事件分支：不打守军，按事件类型分发 ----
		if ev := matchingEvent(m); ev != nil {
			handler := eventHandlers[ev.EventType]
			if handler == nil {
				loggers.Logger.Warn("sweep: no handler for event type",
					zap.Uint64("march_id", info.GetMarchID().Uint64()),
					zap.Int32("event_type", int32(ev.EventType)))
				m.DefeatRecall(mgr) // 未注册类型 → 兜底召回
				return
			}
			if err := handler(mgr, info, ev); err != nil {
				loggers.Logger.Warn("sweep event handler failed",
					zap.Uint64("march_id", info.GetMarchID().Uint64()),
					zap.Error(err))
				m.DefeatRecall(mgr)
				return
			}
			// 事件处理成功，从地块清除事件（事件消费）
			if toInfo := m.GetToMapInfo(); toInfo != nil {
				toInfo.SetOverlayEvent(nil)
			}
			return
		}

		// ---- 守军PvE分支：与目标当前等级守军战斗 ----
		settle := mgr.GetBattleSettleFunc()
		if settle == nil {
			loggers.Logger.Warn("sweep: battle settle func not injected",
				zap.Uint64("march_id", info.GetMarchID().Uint64()))
			m.DefeatRecall(mgr)
			return
		}
		req := buildGuardSettleReq(mgr, info)
		rsp, err := settle(req)
		if err != nil || rsp == nil {
			loggers.Logger.Warn("sweep guard battle settle failed",
				zap.Uint64("march_id", info.GetMarchID().Uint64()),
				zap.Error(err))
			m.DefeatRecall(mgr)
			return
		}
		if !rsp.GetAttackerWin() {
			m.DefeatRecall(mgr) // 战斗失败 → 召回
			return
		}
	})

	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return
		}
		// 结算完成返回（不驻留）
		m.CallBack()
	})

	return m
}

// matchingEvent 返回目标当前匹配的事件（ID 一致且未过期）；nil 表示无事件/不匹配
func matchingEvent(m *marchdos.SingleMarch) *map_events.OverlayEvent {
	info := m.MarchInfo()
	if info == nil {
		return nil
	}
	toInfo := m.GetToMapInfo()
	if toInfo == nil {
		return nil
	}
	ev := toInfo.GetOverlayEvent()
	if ev == nil {
		return nil
	}
	if ev.ExpireTime > 0 && ev.ExpireTime <= time.Now().Unix() {
		return nil // 已过期视为无事件
	}
	if ev.EventID != info.GetTargetEventID() {
		return nil // 事件ID不一致（事件被替换）
	}
	return ev
}

// checkGuardTarget 守军PvE分支：目标当前等级须有守军配置
func checkGuardTarget(mgr *map_managers.MapManager, info *marchs.MarchInfo) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok || toMapInfo == nil {
		return false
	}
	guardFunc := mgr.GetGuardConfigFunc()
	if guardFunc == nil {
		return false
	}
	if slots := guardFunc(cores_declarations.MapLevel(int32(toMapInfo.GetLevel()))); len(slots) == 0 {
		return false
	}
	return true
}

// buildGuardSettleReq 组装守军PvE战斗结算请求（攻击方=行军队伍；防守方=目标当前等级守军NPC）
func buildGuardSettleReq(mgr *map_managers.MapManager, info *marchs.MarchInfo) *pb_battle.BattleSettleReq {
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

	guardSlots := guardFunc(cores_declarations.MapLevel(int32(toMapInfo.GetLevel())))
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
