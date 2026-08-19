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

// realGameconfig 仓库真实 gameconfig.json（tabtoy 导出，与内嵌 gameconfigjson 同源）。
func realGameconfig(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("json", gameconfigFileName))
	if err != nil {
		t.Fatalf("read real gameconfig.json: %v", err)
	}
	return data
}

// writeFile 写文件（测试 helper）。
func writeFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// writeGameconfig 写 temp 目录下的 gameconfig.json。
func writeGameconfig(t *testing.T, dir string, data []byte) {
	t.Helper()
	writeFile(t, dir, gameconfigFileName, data)
}

// replaceOnce 在字节中替换首次出现的 old；未命中直接 Fatal（防止测试悄悄没生效）。
func replaceOnce(t *testing.T, data []byte, old, new string) []byte {
	t.Helper()
	out := strings.Replace(string(data), old, new, 1)
	if out == string(data) {
		t.Fatalf("%q not found in gameconfig.json", old)
	}
	return []byte(out)
}

// TestLoadAll_FromJSON 真实 gameconfig.json 全量加载：索引、版本、内容 hash、跨表校验均生效。
func TestLoadAll_FromJSON(t *testing.T) {
	dir := t.TempDir()
	writeGameconfig(t, dir, realGameconfig(t))

	gc, err := loadAll(dir)
	if err != nil {
		t.Fatalf("loadAll failed: %v", err)
	}
	if gc.Hero.MaxLevel != 100 || gc.Hero.NeedExp(1) != 100 {
		t.Errorf("hero: max_level=%d need_exp1=%d", gc.Hero.MaxLevel, gc.Hero.NeedExp(1))
	}
	if _, ok := gc.Hero.HeroConf(1); !ok {
		t.Error("hero 1 not found")
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
	if gc.filePath != dir {
		t.Errorf("filePath = %q, want %q", gc.filePath, dir)
	}
	if gc.globalVersion < 1 {
		t.Errorf("globalVersion = %d, want >= 1", gc.globalVersion)
	}
	if v := gc.tableVersions[gameconfigFileName]; v == "" {
		t.Error("tableVersions[gameconfig.json] should be non-empty content hash")
	}
	if gc.All() == nil {
		t.Error("All() should return the pb table after load")
	}
}

// TestLoadAll_MissingFile_Fails 单文件缺失 → fail-fast：loadAll/Init 返回 err 且不替换全局配置。
func TestLoadAll_MissingFile_Fails(t *testing.T) {
	dir := t.TempDir() // 无 gameconfig.json

	old := Load()
	if _, err := loadAll(dir); err == nil {
		t.Fatal("loadAll should fail when gameconfig.json missing")
	}
	if err := Init(dir); err == nil {
		t.Fatal("Init should fail when gameconfig.json missing")
	}
	if Load() != old {
		t.Error("defaultConf should be unchanged after failed Init")
	}
}

// TestLoadAll_InvalidJSON_Rollback 坏 JSON → loadAll/Init 返回 err 且不替换全局配置。
func TestLoadAll_InvalidJSON_Rollback(t *testing.T) {
	dir := t.TempDir()
	writeGameconfig(t, dir, []byte(`{invalid json`))

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

// TestLoadAll_ValidateFailure 构建校验失败（全空表 → hero 表空）→ err。
func TestLoadAll_ValidateFailure(t *testing.T) {
	dir := t.TempDir()
	writeGameconfig(t, dir, []byte(`{}`))

	if _, err := loadAll(dir); err == nil {
		t.Fatal("loadAll should fail on validate error")
	}
}

// TestLoadBattle_Subset 轻量初始化：仅 battle+skill，其余 nil。
func TestLoadBattle_Subset(t *testing.T) {
	dir := t.TempDir()
	writeGameconfig(t, dir, realGameconfig(t))

	gc, err := loadBattle(dir)
	if err != nil {
		t.Fatalf("loadBattle failed: %v", err)
	}
	if gc.Battle == nil || gc.Skill == nil {
		t.Fatal("battle+skill should be loaded in battle subset")
	}
	if gc.Hero != nil || gc.Item != nil || gc.Gacha != nil {
		t.Error("non-battle domains should be nil in battle subset")
	}
	if !gc.battleOnly {
		t.Error("battleOnly flag should be true")
	}
	if gc.Battle.Rounds != 8 {
		t.Errorf("battle rounds = %d, want 8", gc.Battle.Rounds)
	}
}

// TestReLoad_FullReloadCycle 基于真实 gameconfig.json 的完整热更周期：
//
//	改值 → reload 成功 + 版本 +1 + 值生效；改坏 → reload 失败 + 保持旧配置。
func TestReLoad_FullReloadCycle(t *testing.T) {
	dir := t.TempDir()
	writeGameconfig(t, dir, realGameconfig(t))

	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	v0 := Load().Version()
	if Load().Hero.MaxLevel != 100 || Load().Battle.Rounds != 8 {
		t.Fatalf("initial load wrong: hero=%d battle_rounds=%d", Load().Hero.MaxLevel, Load().Battle.Rounds)
	}

	// ① 改 rounds 8→9（无跨表依赖，单值替换）→ reload 成功版本+1
	orig, _ := os.ReadFile(filepath.Join(dir, gameconfigFileName))
	writeGameconfig(t, dir, replaceOnce(t, orig, `"rounds": 8`, `"rounds": 9`))

	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad after change failed: %v", err)
	}
	if Load().Version() != v0+1 {
		t.Errorf("version = %d, want %d (bumped on hot reload)", Load().Version(), v0+1)
	}
	if Load().Battle.Rounds != 9 {
		t.Errorf("battle rounds = %d, want 9 (hot reload applied)", Load().Battle.Rounds)
	}

	// ② 改坏 gameconfig.json → ReLoad 失败 → 保持旧配置（rounds 仍 9、版本不变）
	writeGameconfig(t, dir, []byte(`{broken json`))
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
	writeGameconfig(t, dir, realGameconfig(t))
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
	writeGameconfig(t, dir, realGameconfig(t))
	if err := Init(dir); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	before := Load().Version()

	orig, _ := os.ReadFile(filepath.Join(dir, gameconfigFileName))
	writeGameconfig(t, dir, replaceOnce(t, orig, `"rounds": 8`, `"rounds": 7`))

	if err := ReLoad(); err != nil {
		t.Fatalf("ReLoad failed: %v", err)
	}
	if after := Load().Version(); after != before+1 {
		t.Errorf("version = %d, want %d", after, before+1)
	}
	if Load().Battle.Rounds != 7 {
		t.Errorf("battle rounds = %d, want 7 (hot reload applied)", Load().Battle.Rounds)
	}
}
