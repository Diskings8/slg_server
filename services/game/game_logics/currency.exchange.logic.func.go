package game_logics

import (
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/exchange"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// ExchangeResult 货币兑换结果
type ExchangeResult struct {
	RemainFrom int64 // 兑换后来源货币剩余
	Obtain     int64 // 本次获得目标货币数量
}

// CurrencyExchange 货币兑换（一级→二级）
//
//	按配置规则整组兑换：count 需为 rule.FromCount 的整数倍（非倍数直接报错，不做取整/四舍五入）。
//	扣发统一走 ItemChange，产消日志 reason = "exchange"。
func CurrencyExchange(role *game_roles.Role, fromID, toID pb_confs.ItemID, count int64) (*ExchangeResult, error) {
	rule, ok := game_conf.Load().Exchange.GetRule(fromID)
	if !ok {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_CurrencyExchangeConfNotFound, fmt.Sprintf("exchange rule not found: from=%d", fromID))
	}
	if err := validateExchange(rule, toID, count); err != nil {
		return nil, err
	}

	groups := count / rule.FromCount
	obtain := groups * rule.ToCount

	if err := ItemChange(role,
		[]common_declarations.ItemUse{{ItemID: rule.ToID, ItemType: rule.ToType, Count: obtain}},
		[]common_declarations.ItemUse{{ItemID: rule.FromID, ItemType: rule.FromType, Count: count}},
		common_declarations.ReasonExchange); err != nil {
		return nil, err
	}

	return &ExchangeResult{
		RemainFrom: role.GetItems().GetItemCount(int32(rule.FromID)),
		Obtain:     obtain,
	}, nil
}

// validateExchange 校验兑换规则：目标货币匹配 + 数量为正 + 整组倍数（纯函数，便于单测）
func validateExchange(rule *exchange.RuleConfig, toID pb_confs.ItemID, count int64) error {
	if rule.ToID != toID {
		return rpc_results.Error(pb_error_code.ErrorCode_CurrencyExchangeConfNotFound, fmt.Sprintf("exchange target not match: from=%d to=%d", rule.FromID, toID))
	}
	if count <= 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "exchange count must be positive")
	}
	if count%rule.FromCount != 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_CurrencyExchangeCountInvalid, fmt.Sprintf("exchange count must be multiple of %d", rule.FromCount))
	}
	return nil
}
