package item_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_logics"
)

// HandlerUseItem 使用道具 (1000005)
func HandlerUseItem(ctx context.Context, roleID uint64, req *pb_item.UseItemReq, resp *pb_item.UseItemResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	item := req.GetItem()
	if item == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "item is nil")
	}

	// 构造消耗列表
	useItems := []common_declarations.ItemUse{
		{
			ItemID:   pb_confs.ItemID(item.GetConfId()),
			Count:    item.GetCount(),
		},
	}

	// 统一扣道具
	// TODO: 根据道具 ItemType/ItemSubType 执行具体效果
	//       - 对英雄使用 → 加经验、加培养属性等
	//       - 无目标 → 直接使用（资源包、宝箱等）
	if err := game_logics.ItemChange(role, nil, useItems, common_declarations.ReasonUse); err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ItemTypeNormalNotEnough, err.Error())
	}

	// 返回剩余数量
	remain := role.GetItems().GetItemCount(int32(item.GetConfId()))
	resp.Remain = &pb_item.ItemUse{
		ConfId: item.GetConfId(),
		Count:  remain,
	}

	return nil
}
