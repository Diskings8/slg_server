package soldier

import "testing"

// TestSoldierLimit_Embedded 内嵌占位：英雄等级基础 + 兵营等级累计加成
func TestSoldierLimit_Embedded(t *testing.T) {
	c := New()

	cases := []struct {
		name         string
		heroLevel    uint32
		barrackLevel uint32
		want         uint32
	}{
		{"lv1 no barrack", 1, 0, 100},
		{"lv9 no barrack", 9, 0, 100}, // 1~9 级断点均 100
		{"lv10 no barrack", 10, 0, 200},
		{"lv15 no barrack", 15, 0, 200}, // 10~19 级均 200
		{"lv20 no barrack", 20, 0, 350},
		{"lv1 barrack1", 1, 1, 100},   // +0
		{"lv1 barrack2", 1, 2, 150},   // +50
		{"lv1 barrack3", 1, 3, 200},   // +100
		{"lv10 barrack3", 10, 3, 300}, // 200+100
		{"lv20 barrack3", 20, 3, 450}, // 350+100
	}
	for _, tc := range cases {
		got := c.SoldierLimit(tc.heroLevel, tc.barrackLevel)
		if got != tc.want {
			t.Errorf("%s: SoldierLimit(%d,%d)=%d, want %d", tc.name, tc.heroLevel, tc.barrackLevel, got, tc.want)
		}
	}
}

// TestSoldierLimit_JSONLoad JSON 加载后计算与内嵌一致
func TestSoldierLimit_JSONLoad(t *testing.T) {
	jsonData := []byte(`{
		"default_soldier_num": 100,
		"hero_level_caps": [{"level":1,"soldier_num":100},{"level":10,"soldier_num":200},{"level":20,"soldier_num":350}],
		"barrack_level_bonus": [{"level":1,"bonus":0},{"level":2,"bonus":50},{"level":3,"bonus":100}]
	}`)
	c := New()
	if err := c.Load(jsonData); err != nil {
		t.Fatalf("Load err: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate err: %v", err)
	}

	if c.DefaultSoldierNum != 100 {
		t.Errorf("DefaultSoldierNum=%d, want 100", c.DefaultSoldierNum)
	}
	if got := c.SoldierLimit(1, 0); got != 100 {
		t.Errorf("SoldierLimit(1,0)=%d, want 100", got)
	}
	if got := c.SoldierLimit(10, 3); got != 300 {
		t.Errorf("SoldierLimit(10,3)=%d, want 300", got)
	}
	if got := c.SoldierLimit(20, 3); got != 450 {
		t.Errorf("SoldierLimit(20,3)=%d, want 450", got)
	}
}

// TestValidate_Invalid 校验失败场景
func TestValidate_Invalid(t *testing.T) {
	c := New()
	c.DefaultSoldierNum = 0
	if err := c.Validate(); err == nil {
		t.Error("want error for default_soldier_num=0, got nil")
	}

	c = New()
	c.heroCaps = []uint32{}
	c.maxHeroLevel = 0
	if err := c.Validate(); err == nil {
		t.Error("want error for empty hero_caps, got nil")
	}
}
