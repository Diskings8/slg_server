package marchs

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// Team 行军队伍，包含出征的武将和士兵集合，提供存活数、受伤数和战斗能力检查
type Team struct {
	Slots []*pb_battle.TeamSlotInfo
}

func (t *Team) Format2Pb() *pb_battle.TeamInfo {
	return &pb_battle.TeamInfo{
		SlotInfo: t.Slots,
	}
}

// ApplyTeamInfo 用战斗结算快照覆盖队伍 slot 数据。
// battle 节点返回的 rsp 是独立反序列化对象，直接替换整片 slice，无指针别名问题。
// 调用方需已持有 MarchInfo 写锁。
func (t *Team) ApplyTeamInfo(snapshot *pb_battle.TeamInfo) {
	if t == nil || snapshot == nil {
		return
	}
	t.Slots = snapshot.SlotInfo
}

func (t *Team) GetAliveSoliderCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Slots {
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			if si := v.GetHeroInfo().GetSoldierInfo(); si != nil {
				sum += uint64(si.GetCurAliveNum())
			}
		}
	}
	return sum
}

func (t *Team) GetMaxCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Slots {
		if si := v.GetHeroInfo().GetSoldierInfo(); si != nil {
			sum += uint64(si.GetMaxSoldierNum())
		}
	}
	return sum
}

func (t *Team) CheckCanFight() bool {
	for _, v := range t.Slots {
		if v.GetSlotId() == cores_declarations.TeamSlot1 {
			return v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured
		}
	}
	return false
}
