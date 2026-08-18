package game_logics

import (
	"testing"

	"server.slg.com/api/game_conf"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// TestGmSetHeroLevel GM：设等级生效 + 经验清零 + 重算属性
func TestGmSetHeroLevel(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1)

	if err := GmSetHeroLevel(hero, 20); err != nil {
		t.Fatalf("GmSetHeroLevel failed: %v", err)
	}
	if hero.GetLevel() != 20 {
		t.Errorf("level = %d, want 20", hero.GetLevel())
	}
	if hero.GetExp() != 0 {
		t.Errorf("exp = %d, want 0 (cleared)", hero.GetExp())
	}
	// cur_val 已按 20 级重算（攻击 = base + growth×19）
	conf, _ := game_conf.Load().Hero.HeroConf(1)
	wantAtk := conf.Base.Attack + conf.Growth.Attack*19
	if got := hero.Cultivates[0].GetCurVal(); got != wantAtk {
		t.Errorf("attack cur_val = %d, want %d", got, wantAtk)
	}

	// 超上限 → 拒绝
	if err := GmSetHeroLevel(hero, 101); err == nil {
		t.Fatal("want error for level > max")
	}
}

// TestHeroAddExp_LevelUp 恰跨 1 级：level1 + 100exp → level2（NeedExp(1)=100 恰好消耗）
func TestHeroAddExp_LevelUp(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1) // 默认 level1
	level, err := HeroAddExp(hero, 100)
	if err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}
	if level != 2 {
		t.Errorf("level = %d, want 2", level)
	}
	if hero.GetExp() != 0 {
		t.Errorf("exp = %d, want 0 (exactly consumed)", hero.GetExp())
	}
	if hero.GetAttrPoint() != 0 {
		t.Errorf("attr_point = %d, want 0 (not level 10 yet)", hero.GetAttrPoint())
	}
}

// TestHeroAddExp_MultiLevelUp 连升多级：level1 + 300exp → level3（100 + 200）
func TestHeroAddExp_MultiLevelUp(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1)
	level, err := HeroAddExp(hero, 300)
	if err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}
	if level != 3 {
		t.Errorf("level = %d, want 3", level)
	}
	if hero.GetExp() != 0 {
		t.Errorf("exp = %d, want 0", hero.GetExp())
	}
}

// TestHeroAddExp_AttrPointAt10 升到 10 级发自由属性点（FreePointPer10L=5）
func TestHeroAddExp_AttrPointAt10(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1)
	// level1→10 需 NeedExp(1..9) = 100+200+...+900 = 4500
	level, err := HeroAddExp(hero, 4500)
	if err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}
	if level != 10 {
		t.Errorf("level = %d, want 10", level)
	}
	if got := hero.GetAttrPoint(); got != game_conf.Load().Hero.FreePointPer10L {
		t.Errorf("attr_point = %d, want %d", got, game_conf.Load().Hero.FreePointPer10L)
	}
}

// TestHeroAddExp_RefreshCurVal 升级后 cur_val 按新等级刷新（battle 快照用）
func TestHeroAddExp_RefreshCurVal(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1)
	// 创建即初始化：lv1 → cur_val = 基础属性（攻100 拆20）
	if got := hero.Cultivates[0].GetCurVal(); got != 100 {
		t.Errorf("lv1 attack cur_val = %d, want 100", got)
	}
	if got := hero.Cultivates[4].GetCurVal(); got != 20 {
		t.Errorf("lv1 relocation cur_val = %d, want 20", got)
	}

	if _, err := HeroAddExp(hero, 300); err != nil { // level1→3
		t.Fatalf("HeroAddExp failed: %v", err)
	}
	// level3 → base + growth×2：攻120 拆24
	if got := hero.Cultivates[0].GetCurVal(); got != 120 {
		t.Errorf("lv3 attack cur_val = %d, want 120", got)
	}
	if got := hero.Cultivates[4].GetCurVal(); got != 24 {
		t.Errorf("lv3 relocation cur_val = %d, want 24", got)
	}
}

// TestHeroAddExp_MaxLevel 满级后多余经验不触发升级（经验累计但不升级）
func TestHeroAddExp_MaxLevel(t *testing.T) {
	hero := game_roles.NewTest(50001).GetHeroes().AddHero(1)
	hero.SetLevel(game_conf.Load().Hero.MaxLevel)
	hero.SetExp(0)

	level, err := HeroAddExp(hero, 999999)
	if err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}
	if level != game_conf.Load().Hero.MaxLevel {
		t.Errorf("level = %d, want max %d", level, game_conf.Load().Hero.MaxLevel)
	}
	if hero.GetExp() != 999999 {
		t.Errorf("exp = %d, want 999999 (capped at max level)", hero.GetExp())
	}
}

// TestHeroNeedExpTable 逐级经验表读取（含越界）
func TestHeroNeedExpTable(t *testing.T) {
	hc := game_conf.Load().Hero
	cases := []struct {
		level uint32
		want  uint32
	}{
		{1, 100},
		{2, 200},
		{100, 10000},
		{0, 0},   // 越界（level 从 1 起）
		{101, 0}, // 越界（超过 MaxLevel）
	}
	for _, c := range cases {
		if got := hc.NeedExp(c.level); got != c.want {
			t.Errorf("NeedExp(%d) = %d, want %d", c.level, got, c.want)
		}
	}
}
