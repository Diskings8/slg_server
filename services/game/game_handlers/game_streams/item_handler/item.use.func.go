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

// HandlerUseItem 使用道具 (1000005)
//
// 统一扣道具，按道具配置的 effect 分发执行效果：
//   - 对英雄使用（target.hero_id）→ 加英雄经验等
//   - 无目标 → 资源包/货币礼包直接发放
func HandlerUseItem(ctx context.Context, roleID uint64, req *pb_item.UseItemReq, resp *pb_item.UseItemResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	item := req.GetItem()
	if item == nil || item.GetConfId() <= 0 || item.GetCount() <= 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid item")
	}

	if logicErr := game_logics.ApplyItemEffect(role, pb_confs.ItemID(item.GetConfId()), item.GetCount(), req.GetHeroId()); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r // 效果配置不存在/目标无效/道具不足等专属错误码透传
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("use item failed: %s", logicErr.Error()))
	}

	poller.Save() // 打脏标记，异步保存道具/英雄变更

	// 返回剩余数量
	remain := role.GetItems().GetItemCount(item.GetConfId())
	resp.Remain = &pb_item.ItemUse{
		ConfId: item.GetConfId(),
		Count:  remain,
	}
	return nil
}
