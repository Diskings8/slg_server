package hero

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// TestHero_LoadAndQuery 经 pb.Table → NewFromPB 构建后查询能力正常（经验表/英雄属性/派生属性）。
func TestHero_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Hero: []*pb_gameconfig.Hero{
			{MaxLevel: 3, FreePointPer_10L: 5, MaxStarStage: 5, StarPointPer: 5},
		},
		HeroExp: []*pb_gameconfig.HeroExp{
			{Level: 1, Exp: 100}, {Level: 2, Exp: 200}, {Level: 3, Exp: 300},
		},
		HeroAttr: []*pb_gameconfig.HeroAttr{
			{ConfId: 1, BaseAttack: 100, BaseDefense: 80, BaseIntelligence: 60, BaseMovement: 50, BaseRelocation: 20,
				GrowthAttack: 10, GrowthDefense: 8, GrowthIntelligence: 6, GrowthMovement: 5, GrowthRelocation: 2,
				AttackRange: 3},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	if c.NeedExp(1) != 100 || c.NeedExp(2) != 200 {
		t.Errorf("NeedExp(1,2) = %d,%d, want 100,200", c.NeedExp(1), c.NeedExp(2))
	}
	if c.NeedExp(3) != 300 { // 3 = max_level，仍可查（升级到 4 的经验）
		t.Errorf("NeedExp(3) = %d, want 300", c.NeedExp(3))
	}
	if c.NeedExp(4) != 0 { // 越界
		t.Errorf("NeedExp(4) = %d, want 0", c.NeedExp(4))
	}
	hc, ok := c.HeroConf(1)
	if !ok || hc.Base.Attack != 100 {
		t.Errorf("HeroConf(1) = %+v, ok=%v, want attack 100", hc, ok)
	}
	// cur_val = base + growth*(level-1)：lv3 攻 100+10*2=120
	if cur := c.CalcCurVal(1, 3); cur.Attack != 120 {
		t.Errorf("CalcCurVal(1,3).Attack = %d, want 120", cur.Attack)
	}
}

// TestHero_LoadDuplicateKey 主键重复 → NewFromPB 报错（不做静默覆盖）。
func TestHero_LoadDuplicateKey(t *testing.T) {
	_, err := NewFromPB(&pb_gameconfig.Table{
		Hero: []*pb_gameconfig.Hero{
			{MaxLevel: 2, FreePointPer_10L: 5, MaxStarStage: 5, StarPointPer: 5},
		},
		HeroExp: []*pb_gameconfig.HeroExp{
			{Level: 1, Exp: 100}, {Level: 2, Exp: 200},
		},
		HeroAttr: []*pb_gameconfig.HeroAttr{
			{ConfId: 1, BaseAttack: 100, BaseDefense: 80, BaseIntelligence: 60, BaseMovement: 50, BaseRelocation: 20,
				GrowthAttack: 10, GrowthDefense: 8, GrowthIntelligence: 6, GrowthMovement: 5, GrowthRelocation: 2, AttackRange: 3},
			{ConfId: 1, BaseAttack: 200, BaseDefense: 80, BaseIntelligence: 60, BaseMovement: 50, BaseRelocation: 20,
				GrowthAttack: 10, GrowthDefense: 8, GrowthIntelligence: 6, GrowthMovement: 5, GrowthRelocation: 2, AttackRange: 3},
		},
	})
	if err == nil {
		t.Fatal("NewFromPB should fail on duplicate conf_id")
	}
}

// TestHero_ValidateExpNeedMismatch exp_need 长度 ≠ max_level → Validate 报错。
func TestHero_ValidateExpNeedMismatch(t *testing.T) {
	c := New()
	c.MaxLevel = 3
	c.ExpNeed = []uint32{100} // 长度 1 != 3
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when exp_need length != max_level")
	}
}
