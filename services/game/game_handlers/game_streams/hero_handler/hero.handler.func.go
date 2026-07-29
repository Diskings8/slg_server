package hero_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_role_handler"
)

// HandlerHeroList 获取英雄列表 (1000001)
func HandlerHeroList(ctx context.Context, roleID uint64, req *pb_hero.HeroListReq, resp *pb_hero.HeroListResp) rpc_results.ResultI {
	poller, role, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_RoleNotFound, fmt.Sprintf("get role failed: %s", err.DevMsg()))
	}
	defer poller.Release()

	resp.Heroes = role.GetHeroes().Format2Pb()
	return nil
}
