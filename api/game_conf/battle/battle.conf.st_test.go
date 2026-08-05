package battle

import "testing"

// TestInjuryRate 受伤比例：第1回合 85%，每回合 -10%，最低 0
func TestInjuryRate(t *testing.T) {
	c := New()
	cases := []struct {
		round uint32
		want  uint32
	}{
		{1, 85},
		{2, 75},
		{9, 5},  // 85 - 8×10 = 5
		{10, 0}, // 85 - 9×10 < 0 → 0
	}
	for _, cse := range cases {
		if got := c.InjuryRate(cse.round); got != cse.want {
			t.Errorf("InjuryRate(%d) = %d, want %d", cse.round, got, cse.want)
		}
	}
}

// TestBattleRules 基础规则值
func TestBattleRules(t *testing.T) {
	c := New()
	if c.Rounds != 8 {
		t.Errorf("Rounds = %d, want 8", c.Rounds)
	}
	if c.SettlementDeadRate != 10 {
		t.Errorf("SettlementDeadRate = %d, want 10", c.SettlementDeadRate)
	}
	if c.PhysCoeff() != 1.0 || c.MagicCoeff() != 1.0 {
		t.Errorf("收敛系数应为 1.0，实际 phys=%v magic=%v", c.PhysCoeff(), c.MagicCoeff())
	}
	if c.BattleExpCoeff != 5 {
		t.Errorf("BattleExpCoeff = %d, want 5", c.BattleExpCoeff)
	}
}
