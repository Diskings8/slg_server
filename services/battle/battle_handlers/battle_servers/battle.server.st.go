package battle_servers

import (
	"server.slg.com/api/protocol/pb/pb_battle"
)

// BattleServer 战斗结算 RPC 服务（无状态，纯计算）
var BattleServerHandler = &BattleServer{}

type BattleServer struct {
	pb_battle.UnimplementedBattleHandlerServer
}
