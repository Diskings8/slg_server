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

func (t *Team) GetAliveSoliderCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Slots {
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			sum += uint64(v.GetCurAliveNum())
		}
	}
	return sum
}

func (t *Team) GetMaxCount() uint64 {
	var sum = uint64(0)
	for _, v := range t.Slots {
		sum += uint64(v.GetMaxSoldierNum())
	}
	return sum
}

func (t *Team) CheckCanFight() bool {
	for _, v := range t.Slots {
		if v.GetSlotId() == cores_declarations.TeamSlot_1 {
			return v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured
		}
	}
	return false
}
