package gacha

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGacha_LoadAndQuery JSON 加载后抽卡池/掉落组/保底查询正常。
func TestGacha_LoadAndQuery(t *testing.T) {
	c := New()
	data := []byte(`{
  "pools": [
    {
      "pool_id": 1001, "name": "新手池", "ticket_conf_id": 2004,
      "single_ticket": 1, "single_gold": 100, "ten_ticket": 10, "ten_gold": 900,
      "free_daily": true, "half_price": true,
      "guarantee_times": 10, "guarantee_group_id": 3, "first_drop_group_id": 2,
      "wish_heros": [2, 3], "wish_times": 20,
      "drop_groups": [
        {"group_id": 1, "weight": 70, "items": [
          {"reward_conf_id": 1, "is_hero": true, "count": 1, "weight": 40},
          {"reward_conf_id": 2001, "is_hero": false, "count": 5, "weight": 30}
        ]},
        {"group_id": 3, "weight": 5, "items": [
          {"reward_conf_id": 4, "is_hero": true, "count": 1, "weight": 50, "guarantee_reset": true}
        ]}
      ]
    }
  ]
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if c.FileName() != "gacha" {
		t.Errorf("FileName = %q, want gacha", c.FileName())
	}
	if c.Version() == "" {
		t.Error("Version should be non-empty after JSON load")
	}
	pool, ok := c.GetPool(1001)
	if !ok || pool.Name != "新手池" || pool.SingleGold != 100 || pool.GuaranteeTimes != 10 {
		t.Errorf("GetPool(1001) = %+v, ok=%v", pool, ok)
	}
	if len(pool.DropGroups) != 2 || pool.DropGroups[0].Items[0].RewardConfID != 1 {
		t.Errorf("drop groups mismatch: %+v", pool.DropGroups)
	}
	if pool.DropGroups[1].Items[0].GuaranteeReset != true {
		t.Error("guarantee_reset should be true on epic item")
	}
	ids := c.AllPoolIDs()
	if len(ids) != 1 || ids[0] != 1001 {
		t.Errorf("AllPoolIDs = %v, want [1001]", ids)
	}
}

// TestGacha_LoadDuplicateKey pool_id 重复 → Load 报错。
func TestGacha_LoadDuplicateKey(t *testing.T) {
	c := New()
	data := []byte(`{
  "pools": [
    {"pool_id": 1001, "name": "a", "ticket_conf_id": 2004, "single_ticket": 1, "single_gold": 100, "ten_ticket": 10, "ten_gold": 900, "free_daily": true, "half_price": true, "guarantee_times": 10, "guarantee_group_id": 3, "first_drop_group_id": 0, "wish_heros": [], "wish_times": 20, "drop_groups": [{"group_id": 1, "weight": 70, "items": [{"reward_conf_id": 1, "is_hero": true, "count": 1, "weight": 40}]}]},
    {"pool_id": 1001, "name": "b", "ticket_conf_id": 2004, "single_ticket": 1, "single_gold": 100, "ten_ticket": 10, "ten_gold": 900, "free_daily": true, "half_price": true, "guarantee_times": 10, "guarantee_group_id": 3, "first_drop_group_id": 0, "wish_heros": [], "wish_times": 20, "drop_groups": [{"group_id": 1, "weight": 70, "items": [{"reward_conf_id": 1, "is_hero": true, "count": 1, "weight": 40}]}]}
  ]
}`)
	if err := c.Load(data); err == nil {
		t.Fatal("Load should fail on duplicate pool_id")
	}
}

// TestGacha_ValidateGuaranteeGroupMissing 保底组引用不存在的组 → Validate 报错。
func TestGacha_ValidateGuaranteeGroupMissing(t *testing.T) {
	c := New()
	data := []byte(`{
  "pools": [
    {"pool_id": 1001, "name": "a", "ticket_conf_id": 2004, "single_ticket": 1, "single_gold": 100, "ten_ticket": 10, "ten_gold": 900, "free_daily": true, "half_price": true, "guarantee_times": 10, "guarantee_group_id": 99, "first_drop_group_id": 0, "wish_heros": [], "wish_times": 20, "drop_groups": [{"group_id": 1, "weight": 70, "items": [{"reward_conf_id": 1, "is_hero": true, "count": 1, "weight": 40}]}]}
  ]
}`)
	if err := c.Load(data); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if err := c.Validate(); err == nil {
		t.Fatal("Validate should fail when guarantee_group_id not in drop_groups")
	}
}

// TestGacha_RealJSONMatchesEmbedded 仓库 json/gacha.json 与内嵌占位逐值一致。
func TestGacha_RealJSONMatchesEmbedded(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "json", "gacha.json"))
	if err != nil {
		t.Skipf("gacha.json not found, skip: %v", err)
	}
	jc := New()
	if err := jc.Load(data); err != nil {
		t.Fatalf("load real gacha.json: %v", err)
	}
	if err := jc.Validate(); err != nil {
		t.Fatalf("real gacha.json validate: %v", err)
	}
	// 池 1001 关键字段与内嵌一致
	embed, _ := New().GetPool(1001)
	got, ok := jc.GetPool(1001)
	if !ok {
		t.Fatal("pool 1001 not found in json")
	}
	if got.Name != embed.Name || got.SingleGold != embed.SingleGold || got.TenGold != embed.TenGold ||
		got.GuaranteeTimes != embed.GuaranteeTimes || got.GuaranteeGroupID != embed.GuaranteeGroupID ||
		got.FirstDropGroupID != embed.FirstDropGroupID || got.WishTimes != embed.WishTimes {
		t.Errorf("pool 1001 json=%+v embedded=%+v", got, embed)
	}
	if len(got.DropGroups) != len(embed.DropGroups) {
		t.Fatalf("drop groups len json=%d embedded=%d", len(got.DropGroups), len(embed.DropGroups))
	}
	for i := range embed.DropGroups {
		if got.DropGroups[i].Weight != embed.DropGroups[i].Weight {
			t.Errorf("group %d weight json=%d embedded=%d", i, got.DropGroups[i].Weight, embed.DropGroups[i].Weight)
		}
		if len(got.DropGroups[i].Items) != len(embed.DropGroups[i].Items) {
			t.Errorf("group %d items len mismatch", i)
		}
	}
}
