package item

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
)

// TestItem_LoadAndQuery 经 pb.Table → NewFromPB 构建后道具效果查询正常。
func TestItem_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Item: []*pb_gameconfig.Item{
			{ConfId: 1001},
			{ConfId: 2001, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_hero_exp, EffectValue: 100},
			{ConfId: 2002, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_currency, EffectTarget: 100002, EffectValue: 1000},
			{ConfId: 2003, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_item, EffectTarget: 2001, EffectValue: 5},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
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

// TestItem_LoadDuplicateKey 主键重复 → NewFromPB 报错。
func TestItem_LoadDuplicateKey(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Item: []*pb_gameconfig.Item{
			{ConfId: 2001, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_hero_exp, EffectValue: 100},
			{ConfId: 2001, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_currency, EffectTarget: 100002, EffectValue: 1000},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on duplicate conf_id")
	}
}

// TestItem_ValidateNoneFieldsEffectNone 无效果道具带非零字段 → NewFromPB 校验报错。
func TestItem_ValidateNoneFieldsEffectNone(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Item: []*pb_gameconfig.Item{
			{ConfId: 2004, EffectTarget: 100, EffectValue: 5},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail when EffectNone carries non-zero fields")
	}
}

// TestItem_ValidateAddItemRefMissing AddItem 目标道具不存在 → NewFromPB 校验报错。
func TestItem_ValidateAddItemRefMissing(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Item: []*pb_gameconfig.Item{
			{ConfId: 2003, EffectType: pb_gameconfig.Itemeffecttype_itemeffecttype_add_item, EffectTarget: 9999, EffectValue: 5},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail when AddItem target missing")
	}
}

// TestItem_ValidateInvalidEffectType 非法效果枚举 → NewFromPB 校验报错。
func TestItem_ValidateInvalidEffectType(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Item: []*pb_gameconfig.Item{
			{ConfId: 2001, EffectType: 99, EffectValue: 100},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on invalid effect type")
	}
}
