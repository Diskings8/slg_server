package troop

import (
	"os"
	"path/filepath"
	"testing"
)

// TestTroop_Load JSON 加载后字段生效。
func TestTroop_Load(t *testing.T) {
	c := New()
	data := []byte(`{"transform_level": 10, "default_troop_id": 100, "unlock_item_conf": 1001}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.FileName() != "troop" {
		t.Errorf("FileName = %q, want troop", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
	}
	if c.TransformLevel != 10 || c.DefaultTroopID != 100 || c.UnlockItemConf != 1001 {
		t.Errorf("troop = %d/%d/%d", c.TransformLevel, c.DefaultTroopID, c.UnlockItemConf)
	}
}

// TestTroop_ValidateSameIDs default_troop_id == unlock_item_conf → Validate 报错。
func TestTroop_ValidateSameIDs(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"transform_level": 10, "default_troop_id": 100, "unlock_item_conf": 100}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when default_troop_id == unlock_item_conf")
	}
}

// TestTroop_RealJSON 仓库 json/troop.json 可加载且与内嵌一致。
func TestTroop_RealJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "troop.json"))
	if err != nil {
		t.Skipf("troop.json not found, skip: %v", err)
	}
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real troop.json: %v", err)
	}
	if err := jc.Validate(); err != nil {
		t.Fatalf("real troop.json validate: %v", err)
	}
	embed := New()
	if jc.TransformLevel != embed.TransformLevel || jc.DefaultTroopID != embed.DefaultTroopID || jc.UnlockItemConf != embed.UnlockItemConf {
		t.Errorf("troop json=%d/%d/%d embedded=%d/%d/%d",
			jc.TransformLevel, jc.DefaultTroopID, jc.UnlockItemConf,
			embed.TransformLevel, embed.DefaultTroopID, embed.UnlockItemConf)
	}
}
