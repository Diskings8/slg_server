package item_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerItemList 查询背包 (1000042)
//
// 先惰性结算资源地产出（让资源余额反映最新产量），再返回全部道具/货币/资源。
func HandlerItemList(ctx context.Context, roleID uint64, req *pb_item.ItemListReq, resp *pb_item.ItemListResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	if game_logics.SettleRoleResources(role, roleID) {
		poller.Save()
	}

	resp.Items = role.GetItems().Format2Pb()
	return nil
}
