package item_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerCurrencyExchange 货币兑换（一级→二级）(1000035)
//
// 按配置比例整组兑换：扣来源货币 + 发目标货币，返回来源剩余与本次获得。
func HandlerCurrencyExchange(ctx context.Context, roleID uint64, req *pb_item.CurrencyExchangeReq, resp *pb_item.CurrencyExchangeResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	result, logicErr := game_logics.CurrencyExchange(role, pb_confs.ItemID(req.GetFromId()), pb_confs.ItemID(req.GetToId()), req.GetCount())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 规则不存在/数量非法/货币不足等专属错误码透传
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("currency exchange failed: %s", logicErr.Error()))
	}

	poller.Save() // 打脏标记，异步保存货币变更

	resp.Remain = &pb_item.ItemUse{ConfId: req.GetFromId(), Count: result.RemainFrom}
	resp.Obtain = &pb_item.ItemUse{ConfId: req.GetToId(), Count: result.Obtain}
	return nil
}
