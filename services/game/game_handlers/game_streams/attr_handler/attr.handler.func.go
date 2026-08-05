package attr_handler

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_attr"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerAttrList 查询角色属性 (1000034)
func HandlerAttrList(ctx context.Context, roleID uint64, req *pb_attr.AttrListReq, resp *pb_attr.AttrListResp) rpc_results.ResultI {
	// 只读：免锁快照，无需 Release
	role, result := game_roles.GetCopy(roleID)
	if result != nil {
		return result
	}

	resp.Attr = game_logics.AttrGetPb(role)
	return nil
}
