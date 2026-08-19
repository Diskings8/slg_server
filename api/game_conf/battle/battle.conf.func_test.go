package battle

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// TestBattle_LoadAndQuery 经 pb.Table → NewFromPB 构建后标量生效，公式类方法仍按新数据计算。
func TestBattle_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Battle: []*pb_gameconfig.Battle{
			{Rounds: 10, InjuryRateStart: 90, InjuryRateDecay: 15,
				SettlementDeadRate: 12, PhysConverge: 120, MagicConverge: 110, BattleExpCoeff: 6},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	if c.Rounds != 10 || c.InjuryRateStart != 90 || c.BattleExpCoeff != 6 {
		t.Errorf("battle scalars = %d/%d/%d", c.Rounds, c.InjuryRateStart, c.BattleExpCoeff)
	}
	// 公式仍按新数据：第1回合 90%，第2回合 90-15=75%
	if c.InjuryRate(1) != 90 || c.InjuryRate(2) != 75 {
		t.Errorf("InjuryRate(1,2) = %d,%d, want 90,75", c.InjuryRate(1), c.InjuryRate(2))
	}
	if c.PhysCoeff() != 1.2 {
		t.Errorf("PhysCoeff = %v, want 1.2", c.PhysCoeff())
	}
}

// TestBattle_ValidateDecayExceedsStart decay > start → NewFromPB 校验报错。
func TestBattle_ValidateDecayExceedsStart(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Battle: []*pb_gameconfig.Battle{
			{Rounds: 8, InjuryRateStart: 85, InjuryRateDecay: 90,
				SettlementDeadRate: 10, PhysConverge: 100, MagicConverge: 100, BattleExpCoeff: 5},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail when injury_rate_decay > injury_rate_start")
	}
}
