package hero_handler

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// HandlerHeroList 获取英雄列表 (1000001)
func HandlerHeroList(ctx context.Context, roleID uint64, req *pb_hero.HeroListReq, resp *pb_hero.HeroListResp) rpc_results.ResultI {
	// 只读：免锁快照，无需 Release
	role, result := game_roles.GetCopy(roleID)
	if result != nil {
		return result
	}

	resp.Heroes = role.GetHeroes().Format2Pb()
	return nil
}
