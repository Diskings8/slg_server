package battle_logics

// 队伍快照工具 — 8 回合战斗引擎（battle.framework.func.go）与攻城/PvE 共用的快照操作。

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

// relocationVal 拆迁值 = 未受伤英雄的拆迁值之和（快照 Cultivate 组件求和，game 已算好 cur_val）。
func relocationVal(slots []*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, v := range slots {
		if v == nil || v.GetHeroInfo() == nil {
			continue
		}
		if v.GetHeroInfo().GetCurStatus() != pb_hero.Status_Injured {
			sum += uint64(slotAttr(v).Relocation)
		}
	}
	return sum
}
