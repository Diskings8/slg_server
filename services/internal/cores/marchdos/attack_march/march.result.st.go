package attack_march

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/services/internal/cores/marchs"
)

// LayerResult 单层战斗结果（对应 proto OneBattleResult + 运行时数据）
type LayerResult struct {
	Attacker        *pb_battle.BattleSide // 攻击方战果
	Defender        *pb_battle.BattleSide // 防御方战果
	DefeatedMarches []*marchs.MarchInfo  // 被击败的防守方行军（运行时更新用）
	IsOccupied      bool                 // 是否占领
}

// BattleResult 战斗总结果（对应 proto BattleResults + 运行时数据）
type BattleResult struct {
	Layers            []*LayerResult
	WinCountInc       uint32
	FinalAttackerInfo *marchs.MarchInfo // 攻击方最终行军状态
}
