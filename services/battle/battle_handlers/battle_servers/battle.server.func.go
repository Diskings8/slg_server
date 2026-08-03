package battle_servers

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/services/battle/battle_internals/battle_logics"
)

// BattleSettle 战斗结算
//
// worldmap 行军到达时调用，传入攻守双方队伍快照 + 目标建筑信息，
// 由 battle_logics.Settle 完成纯计算，返回逐层战斗结果。
func (s *BattleServer) BattleSettle(ctx context.Context, req *pb_battle.BattleSettleReq) (*pb_battle.BattleSettleRsp, error) {
	if req == nil || req.GetAttackerTeam() == nil || len(req.GetAttackerTeam().GetSlotInfo()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid battle settle request: attacker team required")
	}
	return battle_logics.Settle(req), nil
}
