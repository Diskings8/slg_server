package hero

import (
	"testing"
)

// TestCalcCurVal 基础/成长/未知英雄（星级不乘属性，加点走 add_val_camp 不进 cur_val）
func TestCalcCurVal(t *testing.T) {
	c := New()

	// 基础：英雄1 level1 → cur_val = base
	attr := c.CalcCurVal(1, 1)
	if attr.Attack != 100 || attr.Defense != 80 || attr.Intelligence != 60 ||
		attr.Movement != 50 || attr.Relocation != 20 {
		t.Errorf("base attr = %+v, want {100,80,60,50,20}", attr)
	}

	// 成长：level3 → attack = 100 + 10×2 = 120
	attr = c.CalcCurVal(1, 3)
	if attr.Attack != 120 {
		t.Errorf("level3 attack = %d, want 120", attr.Attack)
	}

	// 未知英雄 → 0
	if attr := c.CalcCurVal(999, 1); attr.Attack != 0 {
		t.Errorf("unknown hero attack = %d, want 0", attr.Attack)
	}
}

// TestNeedExp 逐级经验表（M2 回归）
func TestNeedExp(t *testing.T) {
	c := New()
	if got := c.NeedExp(1); got != 100 {
		t.Errorf("NeedExp(1) = %d, want 100", got)
	}
	if got := c.NeedExp(101); got != 0 {
		t.Errorf("NeedExp(101) = %d, want 0", got)
	}
}
