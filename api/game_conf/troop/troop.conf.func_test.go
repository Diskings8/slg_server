package troop

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// TestTroop_Load 经 pb.Table → NewFromPB 构建后字段生效。
func TestTroop_Load(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Troop: []*pb_gameconfig.Troop{
			{TransformLevel: 10, DefaultTroopId: 100, UnlockItemConf: 1001},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	if c.TransformLevel != 10 || c.DefaultTroopID != 100 || c.UnlockItemConf != 1001 {
		t.Errorf("troop = %d/%d/%d", c.TransformLevel, c.DefaultTroopID, c.UnlockItemConf)
	}
}

// TestTroop_ValidateSameIDs default_troop_id == unlock_item_conf → NewFromPB 校验报错。
func TestTroop_ValidateSameIDs(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Troop: []*pb_gameconfig.Troop{
			{TransformLevel: 10, DefaultTroopId: 100, UnlockItemConf: 100},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail when default_troop_id == unlock_item_conf")
	}
}
