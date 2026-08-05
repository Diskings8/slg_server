package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// exchangeRole 构造测试角色并注入一级货币（钻石）
func exchangeRole(t *testing.T, currency1 int64) *game_roles.Role {
	t.Helper()
	role := game_roles.NewTest(50003)
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.Currency1ConfID, ItemType: pb_confs.ItemTypeCurrency1, Count: currency1,
	}, time.Now().Unix())
	return role
}

func cur1(role *game_roles.Role) int64 {
	return role.GetItems().GetItemCount(int32(pb_confs.Currency1ConfID))
}

func cur2(role *game_roles.Role) int64 {
	return role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID))
}

// TestCurrencyExchange_Success 100 钻石 → 1000 金币（默认规则 1:10），兑换后来源清零
func TestCurrencyExchange_Success(t *testing.T) {
	role := exchangeRole(t, 100)
	result, err := CurrencyExchange(role, pb_confs.Currency1ConfID, pb_confs.Currency2ConfID, 100)
	if err != nil {
		t.Fatalf("CurrencyExchange failed: %v", err)
	}
	if result.Obtain != 1000 {
		t.Errorf("obtain = %d, want 1000", result.Obtain)
	}
	if result.RemainFrom != 0 {
		t.Errorf("remain = %d, want 0", result.RemainFrom)
	}
	if cur1(role) != 0 || cur2(role) != 1000 {
		t.Errorf("currency1=%d currency2=%d, want 0/1000", cur1(role), cur2(role))
	}
}

// TestCurrencyExchange_Success_Remain 150 钻石换 100 → 剩 50 钻石 + 1000 金币
func TestCurrencyExchange_Success_Remain(t *testing.T) {
	role := exchangeRole(t, 150)
	result, err := CurrencyExchange(role, pb_confs.Currency1ConfID, pb_confs.Currency2ConfID, 100)
	if err != nil {
		t.Fatalf("CurrencyExchange failed: %v", err)
	}
	if result.RemainFrom != 50 {
		t.Errorf("remain = %d, want 50", result.RemainFrom)
	}
	if cur1(role) != 50 || cur2(role) != 1000 {
		t.Errorf("currency1=%d currency2=%d, want 50/1000", cur1(role), cur2(role))
	}
}

// TestCurrencyExchange_NotEnough 余额不足（5 < 100）→ 扣除失败，双方货币不变
func TestCurrencyExchange_NotEnough(t *testing.T) {
	role := exchangeRole(t, 5)
	_, err := CurrencyExchange(role, pb_confs.Currency1ConfID, pb_confs.Currency2ConfID, 100)
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_ItemTypeNormalNotEnough) {
		t.Fatalf("err = %v, want ItemTypeNormalNotEnough", err)
	}
	if cur1(role) != 5 || cur2(role) != 0 {
		t.Errorf("currency should unchanged: currency1=%d currency2=%d", cur1(role), cur2(role))
	}
}

// TestCurrencyExchange_RuleNotFound 来源货币无兑换规则
func TestCurrencyExchange_RuleNotFound(t *testing.T) {
	role := exchangeRole(t, 100)
	_, err := CurrencyExchange(role, 9999, pb_confs.Currency2ConfID, 10)
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_CurrencyExchangeConfNotFound) {
		t.Fatalf("err = %v, want CurrencyExchangeConfNotFound", err)
	}
}

// TestCurrencyExchange_TargetNotMatch 目标货币与规则不匹配
func TestCurrencyExchange_TargetNotMatch(t *testing.T) {
	role := exchangeRole(t, 100)
	_, err := CurrencyExchange(role, pb_confs.Currency1ConfID, 8888, 10)
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_CurrencyExchangeConfNotFound) {
		t.Fatalf("err = %v, want CurrencyExchangeConfNotFound", err)
	}
}

// TestValidateExchange_CountInvalid 非整组倍数（自定义规则 FromCount=10）
func TestValidateExchange_CountInvalid(t *testing.T) {
	rule := &exchange.RuleConfig{FromID: 1, ToID: 2, FromCount: 10, ToCount: 100}
	err := validateExchange(rule, 2, 5) // 5 % 10 != 0
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_CurrencyExchangeCountInvalid) {
		t.Fatalf("err = %v, want CurrencyExchangeCountInvalid", err)
	}
	if err := validateExchange(rule, 2, 20); err != nil {
		t.Errorf("validateExchange(20) = %v, want nil", err)
	}
}

// TestValidateExchange_NonPositive 数量非正 → ParamError
func TestValidateExchange_NonPositive(t *testing.T) {
	rule := &exchange.RuleConfig{FromID: 1, ToID: 2, FromCount: 1, ToCount: 10}
	err := validateExchange(rule, 2, 0)
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_ParamError) {
		t.Fatalf("err = %v, want ParamError", err)
	}
}
