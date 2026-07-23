package attack

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchdos"
	"server.slg.com/services/internal/cores/marchs"
)

const teamSlot1 = 1

func init() {
	marchdos.RegisterMarchFactory(cores_declarations.MarchTypeAttack, New)
}

// BattleResult 战斗结算结果
type BattleResult struct {
	AttackerWin  bool   // 攻击方是否胜利
	DefenderWin  bool   // 防守方是否胜利
	AtkTotalLoss uint32 // 攻击方总阵亡
	DefTotalLoss uint32 // 防守方总阵亡
	AtkSurvive   uint32 // 攻击方存活数
	DefSurvive   uint32 // 防守方存活数
}

// New 创建攻击行军执行器
//
// 攻击行军（10001）到达流水线：
//
//	Prepare → 战前合法性校验 → 不合格则直接返回
//	Do      → 战斗结算 → 战损处理 → 占领判定
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

	var battleResult *BattleResult

	// ---- Prepare：战前校验 ----
	m.AddPrepareOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if !checkTargetLegality(mgr, info) {
			info.MarchState = pb_maps_march.MarchState_Back
		}
	})

	// ---- Do：战斗结算 + 战损 + 占领 ----
	m.AddDoOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil {
			return
		}
		if info.GetMarchState() == pb_maps_march.MarchState_Back {
			return
		}

		battleResult = settleBattle(info, m.GetToMapInfo())
		processBattleResult(mgr, info, m.GetToMapInfo(), battleResult)
	})

	// ---- Finish：战报推送 + 事件触发 ----
	m.AddFinishOpt(func(mgr *map_managers.MapManager) {
		info := m.MarchInfo()
		if info == nil || battleResult == nil {
			return
		}

		pushBattleResult(mgr, info, battleResult)
		triggerBattleEvents(mgr, info, battleResult)
	})

	return m
}

// checkTargetLegality 战前目标合法性校验
//
// 校验项：
//   - 目标地块存在
//   - 目标有归属（ownerID > 0）或被建筑覆盖
//   - 目标不在保护期内（TODO：接入保护期数据后实现）
//   - 不攻击盟友（TODO：接入联盟关系后实现）
func checkTargetLegality(mgr *map_managers.MapManager, info *marchs.MarchInfo) bool {
	toMapInfo, ok := mgr.GetMapDataManager().GetMapInfo(info.GetToMapID())
	if !ok {
		return false
	}

	if toMapInfo.GetOwnerID() == 0 && toMapInfo.GetOverlayBuilding() == nil {
		return false
	}

	// TODO: 检查目标是否在保护期内
	// TODO: 校验是否是盟友（需接入联盟数据）

	return true
}

// settleBattle 执行战斗结算
//
// 采用简化的战力对比模型：
//   - 计算攻击方总战力（存活士兵数 × 10 + 英雄等级 × 100）
//   - 计算防守方总战力（驻军战力 + 建筑默认守军）
//   - 按战力比例分配伤亡
//   - 战力高的一方获胜
//
// TODO: 替换为战斗服务 gRPC 调用，接入技能、属性等完整战斗模型
func settleBattle(info *marchs.MarchInfo, defInfo *map_datas.MapInfo) *BattleResult {
	result := &BattleResult{}
	team := info.GetTeam()
	if team == nil || defInfo == nil {
		result.DefenderWin = true
		return result
	}

	atkPower := calcTeamPower(team)
	defPower := calcDefenderPower(defInfo)

	totalPower := atkPower + defPower
	if totalPower == 0 {
		result.DefenderWin = true
		return result
	}

	atkLossRatio := float64(defPower) / float64(totalPower)
	defLossRatio := float64(atkPower) / float64(totalPower)

	atkTotal := team.GetMaxCount()
	defTotal := getDefenderTotalCount(defInfo)

	result.AtkTotalLoss = uint32(float64(atkTotal) * atkLossRatio)
	result.DefTotalLoss = uint32(float64(defTotal) * defLossRatio)

	// 确保不出现负数
	if result.AtkTotalLoss > atkTotal {
		result.AtkTotalLoss = atkTotal
	}
	if result.DefTotalLoss > defTotal {
		result.DefTotalLoss = defTotal
	}

	result.AtkSurvive = atkTotal - result.AtkTotalLoss
	result.DefSurvive = defTotal - result.DefTotalLoss

	if result.AtkSurvive > result.DefSurvive {
		result.AttackerWin = true
	} else {
		result.DefenderWin = true
	}

	return result
}

// calcTeamPower 计算队伍总战力
func calcTeamPower(team *marchs.Team) uint64 {
	var power uint64
	for _, slot := range team.Slots {
		if slot.GetHeroInfo().GetCurStatus() == pb_hero.Status_Injured {
			continue
		}
		power += uint64(slot.GetCurAliveNum()) * 10
		power += uint64(slot.GetHeroInfo().GetCurLevel()) * 100
	}
	return power
}

// calcDefenderPower 计算防守方战力
func calcDefenderPower(defInfo *map_datas.MapInfo) uint64 {
	if defInfo.GetOwnerID() == 0 {
		return 0
	}
	// TODO: 从 marchManage 获取目标地块驻守列表，累加防守方战力
	// TODO: 根据建筑等级/类型动态计算守军战力
	return 100
}

// getDefenderTotalCount 获取防守方总兵力
func getDefenderTotalCount(defInfo *map_datas.MapInfo) uint32 {
	if defInfo.GetOwnerID() == 0 {
		return 0
	}
	// TODO: 从 marchManage 获取驻守列表，累加兵力
	return 100
}

// processBattleResult 处理战斗结果
func processBattleResult(mgr *map_managers.MapManager, info *marchs.MarchInfo, toMapInfo *map_datas.MapInfo, result *BattleResult) {
	if result == nil || toMapInfo == nil {
		return
	}

	if !info.TryLock() {
		return
	}
	info.MarchState = pb_maps_march.MarchState_Battle
	info.Unlock()

	if result.AttackerWin {
		occupyTile(mgr, info, toMapInfo)
	} else {
		info.MarchState = pb_maps_march.MarchState_Back
	}

	mgr.GetMarchManage().Save(info)
}

// occupyTile 占领地块
func occupyTile(mgr *map_managers.MapManager, info *marchs.MarchInfo, toMapInfo *map_datas.MapInfo) {
	if toMapInfo == nil {
		return
	}

	toMapInfo.Lock()
	defer toMapInfo.Unlock()

	if toMapInfo.GetOverlayBuilding() != nil {
		toMapInfo.GetOverlayBuilding().AfterFree(time.Now())
	}

	toMapInfo.Occupy(info.FromRoleID)
	mgr.GetMapDataManager().Save(toMapInfo)
}

// pushBattleResult 推送战斗结果
func pushBattleResult(mgr *map_managers.MapManager, info *marchs.MarchInfo, result *BattleResult) {
	mgr.UpdateMarchPush(info)
	mgr.UpdateMapPush(info.GetToMapID(), info.GetFromMapID())
}

// triggerBattleEvents 触发战斗事件（预留）
func triggerBattleEvents(mgr *map_managers.MapManager, info *marchs.MarchInfo, result *BattleResult) {
}
