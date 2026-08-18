package game_logics

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// TestMarchBuildTeamOneBasedSlots 出征队伍槽位 1 基（1=大营）+ 攻击距离从配置填充
func TestMarchBuildTeamOneBasedSlots(t *testing.T) {
	role := game_roles.NewTest(60001)
	hero := role.GetHeroes().AddHero(1) // 英雄配置1：AttackRange=3
	formation := role.GetFormations().CreateFormation(role.ID, 0)
	formation.HeroSlots = []*pb_maps_march.HeroSlot{
		{HeroId: hero.ID, SoldierNum: 100},
		{HeroId: hero.ID, SoldierNum: 200},
	}

	slots, res := MarchBuildTeam(role, &pb_maps_march.MarchCreateReq{FormationId: formation.ID})
	if res != nil {
		t.Fatalf("MarchBuildTeam failed: %v", res)
	}
	if len(slots) != 2 {
		t.Fatalf("期望 2 个槽位，实际 %d", len(slots))
	}
	// 1 基槽位：1=大营
	if slots[0].GetSlotId() != 1 {
		t.Errorf("slot[0] = %d, want 1（大营）", slots[0].GetSlotId())
	}
	if slots[1].GetSlotId() != 2 {
		t.Errorf("slot[1] = %d, want 2", slots[1].GetSlotId())
	}
	// 攻击距离从 hero 配置填充（写入 HeroInfo）
	if slots[0].GetHeroInfo().GetAttackRange() != 3 {
		t.Errorf("attack_range = %d, want 3", slots[0].GetHeroInfo().GetAttackRange())
	}
}
