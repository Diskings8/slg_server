package battle

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBattle_LoadAndQuery JSON 加载后标量生效，公式类方法仍按新数据计算。
func TestBattle_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "rounds": 10, "injury_rate_start": 90, "injury_rate_decay": 15,
  "settlement_dead_rate": 12, "phys_converge": 120, "magic_converge": 110, "battle_exp_coeff": 6
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.FileName() != "battle" {
		t.Errorf("FileName = %q, want battle", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
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

// TestBattle_ValidateDecayExceedsStart decay > start → Validate 报错。
func TestBattle_ValidateDecayExceedsStart(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"rounds":8,"injury_rate_start":85,"injury_rate_decay":90,"settlement_dead_rate":10,"phys_converge":100,"magic_converge":100,"battle_exp_coeff":5}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when injury_rate_decay > injury_rate_start")
	}
}

// TestBattle_RealJSON 仓库 json/battle.json 可加载且与内嵌一致。
func TestBattle_RealJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "battle.json"))
	if err != nil {
		t.Skipf("battle.json not found, skip: %v", err)
	}
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real battle.json: %v", err)
	}
	if err := jc.Validate(); err != nil {
		t.Fatalf("real battle.json validate: %v", err)
	}
	embed := New()
	if jc.Rounds != embed.Rounds || jc.InjuryRateStart != embed.InjuryRateStart || jc.PhysConverge != embed.PhysConverge {
		t.Errorf("battle json=%+v embedded=%+v", *jc, *embed)
	}
}
