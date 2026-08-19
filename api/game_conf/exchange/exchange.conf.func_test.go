package exchange

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
)

// TestExchange_LoadAndQuery 经 pb.Table → NewFromPB 构建后兑换规则查询正常。
func TestExchange_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(&pb_gameconfig.Table{
		Exchange: []*pb_gameconfig.Exchange{
			{FromId: 100001, FromType: 1, ToId: 100002, ToType: 2, FromCount: 1, ToCount: 10},
		},
	})
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	rule, ok := c.GetRule(pb_confs.Currency1ConfID)
	if !ok || rule.ToID != pb_confs.Currency2ConfID || rule.FromCount != 1 || rule.ToCount != 10 {
		t.Errorf("GetRule = %+v, ok=%v", rule, ok)
	}
	if _, ok := c.GetRule(9999); ok {
		t.Error("GetRule(9999) should be false")
	}
}

// TestExchange_LoadDuplicateKey from_id 重复 → NewFromPB 报错。
func TestExchange_LoadDuplicateKey(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Exchange: []*pb_gameconfig.Exchange{
			{FromId: 100001, FromType: 1, ToId: 100002, ToType: 2, FromCount: 1, ToCount: 10},
			{FromId: 100001, FromType: 1, ToId: 100002, ToType: 2, FromCount: 10, ToCount: 100},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail on duplicate from_id")
	}
}

// TestExchange_ValidateCurrency1ToCurrency2 一级货币兑换到非二级货币 → NewFromPB 校验报错。
func TestExchange_ValidateCurrency1ToCurrency2(t *testing.T) {
	if _, err := NewFromPB(&pb_gameconfig.Table{
		Exchange: []*pb_gameconfig.Exchange{
			{FromId: 100001, FromType: 1, ToId: 100001, ToType: 1, FromCount: 1, ToCount: 10},
		},
	}); err == nil {
		t.Fatal("NewFromPB should fail when currency1 exchanges to non-currency2")
	}
}
