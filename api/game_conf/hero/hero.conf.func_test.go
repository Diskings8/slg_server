package hero

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHero_LoadAndQuery JSON 加载后查询能力正常（经验表/英雄属性/派生属性）。
func TestHero_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "max_level": 3,
  "free_point_per_10l": 5,
  "max_star_stage": 5,
  "star_point_per": 5,
  "exp_need": [100, 200, 300],
  "heroes": [
    {"conf_id": 1, "base": {"attack": 100, "defense": 80, "intelligence": 60, "movement": 50, "relocation": 20}, "growth": {"attack": 10, "defense": 8, "intelligence": 6, "movement": 5, "relocation": 2}, "attack_range": 3}
  ]
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if c.FileName() != "hero" {
		t.Errorf("FileName = %q, want hero", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
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

// TestHero_LoadDuplicateKey 主键重复 → Load 报错（不做静默覆盖）。
func TestHero_LoadDuplicateKey(t *testing.T) {
	c := New()
	data := []byte(`{
  "max_level": 2,
  "free_point_per_10l": 5,
  "max_star_stage": 5,
  "star_point_per": 5,
  "exp_need": [100, 200],
  "heroes": [
    {"conf_id": 1, "base": {"attack": 100, "defense": 80, "intelligence": 60, "movement": 50, "relocation": 20}, "growth": {"attack": 10, "defense": 8, "intelligence": 6, "movement": 5, "relocation": 2}, "attack_range": 3},
    {"conf_id": 1, "base": {"attack": 200, "defense": 80, "intelligence": 60, "movement": 50, "relocation": 20}, "growth": {"attack": 10, "defense": 8, "intelligence": 6, "movement": 5, "relocation": 2}, "attack_range": 3}
  ]
}`)
	if err := c.Load(data); err == nil {
		t.Fatal("Load should fail on duplicate conf_id")
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

// TestHero_RealJSONMatchesEmbedded 仓库 json/hero.json 与内嵌占位逐值一致（开启 JSON 后行为不变）。
func TestHero_RealJSONMatchesEmbedded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "hero.json"))
	if err != nil {
		t.Skipf("hero.json not found, skip: %v", err)
	}
	c := New() // 内嵌占位
	embedMaxLevel := c.MaxLevel
	embedNeed1 := c.NeedExp(1)
	embedHero1Atk := mustHeroBase(t, c, 1).Attack

	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real hero.json: %v", err)
	}
	if jc.MaxLevel != embedMaxLevel {
		t.Errorf("max_level json=%d embedded=%d", jc.MaxLevel, embedMaxLevel)
	}
	if jc.NeedExp(1) != embedNeed1 {
		t.Errorf("NeedExp(1) json=%d embedded=%d", jc.NeedExp(1), embedNeed1)
	}
	if got := mustHeroBase(t, jc, 1).Attack; got != embedHero1Atk {
		t.Errorf("hero1 base attack json=%d embedded=%d", got, embedHero1Atk)
	}
}

func mustHeroBase(t *testing.T, c *Conf, id int32) HeroAttr {
	t.Helper()
	hc, ok := c.HeroConf(id)
	if !ok {
		t.Fatalf("hero %d not found", id)
	}
	return hc.Base
}
