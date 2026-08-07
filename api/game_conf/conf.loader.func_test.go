package game_conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
)

// TestMain 初始化日志（game_conf 生产代码在加载失败/热更时记录日志，需要非 nil Logger）。
func TestMain(m *testing.M) {
	loggers.Init()
	os.Exit(m.Run())
}

// validHeroJSON 返回一份合法 hero.json（max_level=3，三档经验）。
//
// 含英雄 1、2：内嵌兜底 skill 收藏（101→英雄1×5+英雄2×3）跨表校验依赖两者存在。
func validHeroJSON() []byte {
	return []byte(`{
  "max_level": 3,
  "free_point_per_10l": 5,
  "max_star_stage": 5,
  "star_point_per": 5,
  "exp_need": [100, 200, 300],
  "heroes": [
    {"conf_id": 1, "base": {"attack": 100, "defense": 80, "intelligence": 60, "movement": 50, "relocation": 20}, "growth": {"attack": 10, "defense": 8, "intelligence": 6, "movement": 5, "relocation": 2}, "attack_range": 3},
    {"conf_id": 2, "base": {"attack": 120, "defense": 90, "intelligence": 70, "movement": 55, "relocation": 25}, "growth": {"attack": 12, "defense": 9, "intelligence": 7, "movement": 6, "relocation": 3}, "attack_range": 4}
  ]
}`)
}

func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// TestLoadAll_FromJSON JSON 加载成功：索引、版本、内容 hash 均生效。
func TestLoadAll_FromJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hero.json", validHeroJSON())

	gc, err := loadAll(dir)
	if err != nil {
		t.Fatalf("loadAll failed: %v", err)
	}
	if gc.Hero.MaxLevel != 3 {
		t.Errorf("max_level = %d, want 3", gc.Hero.MaxLevel)
	}
	if got := gc.Hero.NeedExp(1); got != 100 {
		t.Errorf("NeedExp(1) = %d, want 100", got)
	}
	hc, ok := gc.Hero.HeroConf(1)
	if !ok || hc.Base.Attack != 100 {
		t.Errorf("HeroConf(1) = %+v, ok=%v, want attack 100", hc, ok)
	}
	if gc.filePath != dir {
		t.Errorf("filePath = %q, want %q", gc.filePath, dir)
	}
	if gc.globalVersion < 1 {
		t.Errorf("globalVersion = %d, want >= 1", gc.globalVersion)
	}
	if v := gc.tableVersions["hero"]; v == "" {
		t.Error("tableVersions[hero] should be non-empty content hash")
	}
	if v := gc.Hero.Version(); v == "" {
		t.Error("hero.Version() should be non-empty after JSON load")
	}
}

// TestLoadAll_MissingFileKeepsEmbedded 未迁移的表缺失 JSON 时保留 Go 内嵌。
func TestLoadAll_MissingFileKeepsEmbedded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hero.json", validHeroJSON())

	gc, err := loadAll(dir)
	if err != nil {
		t.Fatalf("loadAll failed: %v", err)
	}
	// skill/item 未迁移（无 JSON 文件）→ 保持内嵌，访问方法可用
	if gc.Skill == nil || gc.Item == nil {
		t.Fatal("embedded fallback conf should not be nil")
	}
	if _, ok := gc.Item.Get(2001); !ok {
		t.Error("embedded item 2001 should be reachable")
	}
	if _, ok := gc.tableVersions["skill"]; ok {
		t.Error("skill should not have table version (not migrated)")
	}
}

// TestLoadAll_InvalidJSON_Rollback 坏 JSON → loadAll 返回 err 且不替换全局配置。
func TestLoadAll_InvalidJSON_Rollback(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hero.json", []byte(`{invalid json`))

	old := Load()
	if _, err := loadAll(dir); err == nil {
		t.Fatal("loadAll should fail on invalid json")
	}
	if err := Init(dir); err == nil {
		t.Fatal("Init should fail on invalid json")
	}
	if Load() != old {
		t.Error("defaultConf should be unchanged after failed Init")
	}
}

// TestLoadAll_ValidateFailure 校验失败（exp_need 长度 ≠ max_level）→ err。
func TestLoadAll_ValidateFailure(t *testing.T) {
	dir := t.TempDir()
	bad := `{"max_level":3,"free_point_per_10l":5,"max_star_stage":5,"star_point_per":5,"exp_need":[100],"heroes":[]}`
	writeFile(t, dir, "hero.json", []byte(bad))

	if _, err := loadAll(dir); err == nil {
		t.Fatal("loadAll should fail on validate error")
	}
}

// TestLoadAll_RealJSONDir 从仓库 json/ 目录全量加载（全部表 + 跨表校验），验证 JSON 路径端到端可用。
//
// 覆盖：全部表 JSON 化、跨表校验通过、各域索引可用（升级/技能/道具/抽卡/兑换）。
func TestLoadAll_RealJSONDir(t *testing.T) {
	gc, err := loadAll("json")
	if err != nil {
		t.Fatalf("loadAll(json) failed: %v", err)
	}
	// 全部表 JSON 加载（content hash 非空）
	for _, tbl := range []string{"battle", "hero", "skill", "item", "troop", "exchange", "gacha", "guard", "soldier"} {
		if _, ok := gc.tableVersions[tbl]; !ok {
			t.Errorf("table %s not JSON-loaded", tbl)
		}
	}
	// 抽查各域
	if gc.Hero.MaxLevel != 100 || gc.Hero.NeedExp(1) != 100 {
		t.Errorf("hero: max_level=%d need_exp1=%d", gc.Hero.MaxLevel, gc.Hero.NeedExp(1))
	}
	if _, ok := gc.Skill.GetSkillConf(101); !ok {
		t.Error("skill 101 not found")
	}
	if _, ok := gc.Item.Get(1001); !ok {
		t.Error("item 1001 (troop unlock ref) not found")
	}
	if pool, ok := gc.Gacha.GetPool(1001); !ok || pool.SingleGold != 100 || len(pool.DropGroups) != 3 {
		t.Errorf("gacha pool 1001 wrong: %+v ok=%v", pool, ok)
	}
	if _, ok := gc.Exchange.GetRule(pb_confs.Currency1ConfID); !ok {
		t.Error("exchange rule for currency1 missing")
	}
	if gc.Troop.UnlockItemConf != 1001 {
		t.Errorf("troop unlock_item_conf = %d, want 1001", gc.Troop.UnlockItemConf)
	}
}

// TestReLoad_FullReloadCycle 基于真实 7 表 json 目录的完整热更周期：
//
//	改值 → reload 成功 + 版本 +1 + 值生效；改坏 → reload 失败 + 保持旧配置。
func TestReLoad_FullReloadCycle(t *testing.T) {
	// 复制真实 json/ 到临时目录，避免污染仓库配置
	src := filepath.Join("json")
	tmp := t.TempDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("read json dir: %v", err)
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		writeFile(t, tmp, e.Name(), data)
	}

	if err := Init(tmp); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	v0 := Load().Version()
	if Load().Hero.MaxLevel != 100 || Load().Battle.Rounds != 8 {
		t.Fatalf("initial load wrong: hero=%d battle_rounds=%d", Load().Hero.MaxLevel, Load().Battle.Rounds)
	}

	// ① 改 battle.json rounds 8→9（无跨表依赖，单值替换）→ reload 成功版本+1
	battlePath := filepath.Join(tmp, "battle.json")
	battleData, _ := os.ReadFile(battlePath)
	modified := strings.Replace(string(battleData), `"rounds": 8`, `"rounds": 9`, 1)
	writeFile(t, tmp, "battle.json", []byte(modified))

	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad after change failed: %v", err)
	}
	if Load().Version() != v0+1 {
		t.Errorf("version = %d, want %d (bumped on hot reload)", Load().Version(), v0+1)
	}
	if Load().Battle.Rounds != 9 {
		t.Errorf("battle rounds = %d, want 9 (hot reload applied)", Load().Battle.Rounds)
	}

	// ② 改坏 battle.json → ReLoad 失败 → 保持旧配置（rounds 仍 9、版本不变）
	writeFile(t, tmp, "battle.json", []byte(`{broken json`))
	if err := ReLoad(); err == nil {
		t.Fatal("ReLoad should fail on corrupt json")
	}
	if Load().Battle.Rounds != 9 {
		t.Errorf("battle rounds = %d, want 9 (kept after failed reload)", Load().Battle.Rounds)
	}
	if Load().Version() != v0+1 {
		t.Errorf("version = %d, want %d (unchanged after failed reload)", Load().Version(), v0+1)
	}
}

// TestReLoad_NoPath 无 JSON 路径时 ReLoad 跳过且不 panic。
func TestReLoad_NoPath(t *testing.T) {
	_ = InitDefault() // filePath = ""
	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad with empty path should return nil, got %v", err)
	}
}

// TestReLoad_ContentUnchanged 内容未变 → 跳过原子替换，版本不变。
func TestReLoad_ContentUnchanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hero.json", validHeroJSON())
	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	before := Load().Version()

	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad failed: %v", err)
	}
	if after := Load().Version(); after != before {
		t.Errorf("version changed %d -> %d, want unchanged (content same)", before, after)
	}
}

// TestReLoad_ContentChanged 内容变化 → 热更成功，版本 +1。
func TestReLoad_ContentChanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hero.json", validHeroJSON())
	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	before := Load().Version()

	changed := `{"max_level":4,"free_point_per_10l":5,"max_star_stage":5,"star_point_per":5,"exp_need":[100,200,300,400],"heroes":[{"conf_id":1,"base":{"attack":100,"defense":80,"intelligence":60,"movement":50,"relocation":20},"growth":{"attack":10,"defense":8,"intelligence":6,"movement":5,"relocation":2},"attack_range":3},{"conf_id":2,"base":{"attack":120,"defense":90,"intelligence":70,"movement":55,"relocation":25},"growth":{"attack":12,"defense":9,"intelligence":7,"movement":6,"relocation":3},"attack_range":4}]}`
	writeFile(t, dir, "hero.json", []byte(changed))

	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad failed: %v", err)
	}
	if after := Load().Version(); after != before+1 {
		t.Errorf("version = %d, want %d", after, before+1)
	}
	if Load().Hero.MaxLevel != 4 {
		t.Errorf("hero max_level = %d, want 4 (hot reload applied)", Load().Hero.MaxLevel)
	}
}
