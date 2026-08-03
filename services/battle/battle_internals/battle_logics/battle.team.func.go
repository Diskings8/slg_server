package battle_logics

// 队伍快照工具 — 迁移自 cores/marchdos/attack_march/march.battle.func.go，
// 把对 *marchs.Team / *marchs.MarchInfo 的操作改为对 pb_battle 快照操作。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"

	"google.golang.org/protobuf/proto"
)

// cloneSlots 深拷贝队伍快照，保证结算不污染入参、层间不互相别名。
func cloneSlots(slots []*pb_battle.TeamSlotInfo) []*pb_battle.TeamSlotInfo {
	if slots == nil {
		return nil
	}
	out := make([]*pb_battle.TeamSlotInfo, 0, len(slots))
	for _, s := range slots {
		if s == nil {
			out = append(out, nil)
			continue
		}
		out = append(out, proto.Clone(s).(*pb_battle.TeamSlotInfo))
	}
	return out
}

// aliveSoldierCount 战力 = 未受伤英雄的存活士兵数之和（对齐 Team.GetAliveSoliderCount）。
func aliveSoldierCount(slots []*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, v := range slots {
		if v == nil {
			continue
		}
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			sum += uint64(v.GetCurAliveNum())
		}
	}
	return sum
}

// relocationVal 拆迁值 = 未受伤英雄的拆迁值之和（对齐 MarchInfo.GetRelocationVal）。
func relocationVal(slots []*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, v := range slots {
		if v == nil {
			continue
		}
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			sum += uint64(v.GetHeroInfo().GetAttrRelocation().GetCurVal())
		}
	}
	return sum
}

// applyLossesToSlots 按比例减少各 slot 存活数，受伤英雄跳过（对齐 applyLossesToTeam）。
func applyLossesToSlots(slots []*pb_battle.TeamSlotInfo, beforePower, afterPower uint64) {
	if beforePower == 0 {
		return
	}
	ratio := float64(afterPower) / float64(beforePower)
	for _, slot := range slots {
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

// aliveSoldierCountTeams 多队伍（防守方各行军）总存活士兵数
func aliveSoldierCountTeams(teams [][]*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, team := range teams {
		sum += aliveSoldierCount(team)
	}
	return sum
}

// applyDefenderLosses 按比例减少防守方各队伍 slot 存活数，受伤英雄跳过。
// 多轮战斗中防守方也会承伤。
func applyDefenderLosses(teams [][]*pb_battle.TeamSlotInfo, beforePower, afterPower uint64) {
	if beforePower == 0 {
		return
	}
	ratio := float64(afterPower) / float64(beforePower)
	for _, team := range teams {
		for _, slot := range team {
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

// cloneSlotsTeams 深拷贝多队伍快照（展平为单层 slots，供战报展示）
func cloneSlotsTeams(teams [][]*pb_battle.TeamSlotInfo) []*pb_battle.TeamSlotInfo {
	var out []*pb_battle.TeamSlotInfo
	for _, team := range teams {
		out = append(out, cloneSlots(team)...)
	}
	return out
}
