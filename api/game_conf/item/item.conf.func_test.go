package item

import (
	"os"
	"path/filepath"
	"testing"

	"server.slg.com/api/protocol/pb_confs"
)

// TestItem_LoadAndQuery JSON 加载后道具效果查询正常。
func TestItem_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "items": [
    {"conf_id": 1001, "effect": {"type": 0, "target": 0, "value": 0}},
    {"conf_id": 2001, "effect": {"type": 1, "target": 0, "value": 100}},
    {"conf_id": 2002, "effect": {"type": 2, "target": 100002, "value": 1000}},
    {"conf_id": 2003, "effect": {"type": 3, "target": 2001, "value": 5}}
  ]
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.FileName() != "item" {
		t.Errorf("FileName = %q, want item", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
	}
	ic, ok := c.Get(2002)
	if !ok || ic.Effect.Type != EffectAddCurrency || ic.Effect.Target != int32(pb_confs.Currency2ConfID) || ic.Effect.Value != 1000 {
		t.Errorf("Get(2002) = %+v, ok=%v", ic, ok)
	}
	if !c.Has(1001) {
		t.Error("Has(1001) should be true")
	}
	if c.Has(9999) {
		t.Error("Has(9999) should be false")
	}
}

// TestItem_LoadDuplicateKey 主键重复 → Load 报错。
func TestItem_LoadDuplicateKey(t *testing.T) {
	c := New()
	data := []byte(`{
  "items": [
    {"conf_id": 2001, "effect": {"type": 1, "target": 0, "value": 100}},
    {"conf_id": 2001, "effect": {"type": 2, "target": 100002, "value": 1000}}
  ]
}`)
	if err := c.Load(data); err == nil {
		t.Fatal("Load should fail on duplicate conf_id")
	}
}

// TestItem_ValidateNoneFieldsEffectNone 无效果道具带非零字段 → Validate 报错。
func TestItem_ValidateNoneFieldsEffectNone(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"items":[{"conf_id":2004,"effect":{"type":0,"target":100,"value":5}}]}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when EffectNone carries non-zero fields")
	}
}

// TestItem_ValidateAddItemRefMissing AddItem 目标道具不存在 → Validate 报错。
func TestItem_ValidateAddItemRefMissing(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"items":[{"conf_id":2003,"effect":{"type":3,"target":9999,"value":5}}]}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when AddItem target missing")
	}
}

// TestItem_ValidateInvalidEffectType 非法效果枚举 → Validate 报错。
func TestItem_ValidateInvalidEffectType(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"items":[{"conf_id":2001,"effect":{"type":99,"target":0,"value":100}}]}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail on invalid effect type")
	}
}

// TestItem_RealJSONMatchesEmbedded 仓库 json/item.json 与内嵌占位逐值一致（含补录的 1001）。
func TestItem_RealJSONMatchesEmbedded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "item.json"))
	if err != nil {
		t.Skipf("item.json not found, skip: %v", err)
	}
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real item.json: %v", err)
	}
	if err := jc.Validate(); err != nil {
		t.Fatalf("real item.json validate: %v", err)
	}
	// 内嵌 2001~2004 与 JSON 一致
	for _, id := range []pb_confs.ItemID{2001, 2002, 2003, 2004} {
		embed, _ := New().Get(id)
		got, ok := jc.Get(id)
		if !ok || got != embed {
			t.Errorf("item %d json=%+v embedded=%+v", id, got, embed)
		}
	}
	if !jc.Has(1001) {
		t.Error("item.json should include conf_id 1001 (troop unlock reference)")
	}
}
