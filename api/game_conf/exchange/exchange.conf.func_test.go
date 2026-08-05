package exchange

import (
	"os"
	"path/filepath"
	"testing"

	"server.slg.com/api/protocol/pb_confs"
)

// TestExchange_LoadAndQuery JSON 加载后兑换规则查询正常。
func TestExchange_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "rules": [
    {"from_id": 100001, "from_type": 1, "to_id": 100002, "to_type": 2, "from_count": 1, "to_count": 10}
  ]
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.FileName() != "exchange" {
		t.Errorf("FileName = %q, want exchange", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
	}
	rule, ok := c.GetRule(pb_confs.Currency1ConfID)
	if !ok || rule.ToID != pb_confs.Currency2ConfID || rule.FromCount != 1 || rule.ToCount != 10 {
		t.Errorf("GetRule = %+v, ok=%v", rule, ok)
	}
	if _, ok := c.GetRule(9999); ok {
		t.Error("GetRule(9999) should be false")
	}
}

// TestExchange_LoadDuplicateKey from_id 重复 → Load 报错。
func TestExchange_LoadDuplicateKey(t *testing.T) {
	c := New()
	data := []byte(`{
  "rules": [
    {"from_id": 100001, "from_type": 1, "to_id": 100002, "to_type": 2, "from_count": 1, "to_count": 10},
    {"from_id": 100001, "from_type": 1, "to_id": 100002, "to_type": 2, "from_count": 10, "to_count": 100}
  ]
}`)
	if err := c.Load(data); err == nil {
		t.Fatal("Load should fail on duplicate from_id")
	}
}

// TestExchange_ValidateCurrency1ToCurrency2 一级货币兑换到非二级货币 → Validate 报错。
func TestExchange_ValidateCurrency1ToCurrency2(t *testing.T) {
	c := New()
	if err := c.Load([]byte(`{"rules":[{"from_id":100001,"from_type":1,"to_id":100001,"to_type":1,"from_count":1,"to_count":10}]}`)); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when currency1 exchanges to non-currency2")
	}
}

// TestExchange_RealJSON 仓库 json/exchange.json 可加载且与内嵌一致。
func TestExchange_RealJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "exchange.json"))
	if err != nil {
		t.Skipf("exchange.json not found, skip: %v", err)
	}
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real exchange.json: %v", err)
	}
	if err := jc.Validate(); err != nil {
		t.Fatalf("real exchange.json validate: %v", err)
	}
	embed, _ := New().GetRule(pb_confs.Currency1ConfID)
	got, ok := jc.GetRule(pb_confs.Currency1ConfID)
	if !ok || *got != *embed {
		t.Errorf("exchange json=%+v embedded=%+v, ok=%v", got, embed, ok)
	}
}
