package battle

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建战斗规则（battle 表单行）。
//
// 迁移原 Load 的「局部构建 + 末尾提交 + 校验」：构造失败返回 err，不产生半更新。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Battle) == 0 {
		return nil, fmt.Errorf("battle table empty")
	}
	pb := t.Battle[0]
	c := &Conf{
		Rounds:             pb.Rounds,
		InjuryRateStart:    pb.InjuryRateStart,
		InjuryRateDecay:    pb.InjuryRateDecay,
		SettlementDeadRate: pb.SettlementDeadRate,
		PhysConverge:       pb.PhysConverge,
		MagicConverge:      pb.MagicConverge,
		BattleExpCoeff:     pb.BattleExpCoeff,
	}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("battle validate: %w", err)
	}
	return c, nil
}

// New 构造战斗规则配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	t, err := gameconfigjson.Table()
	if err != nil {
		panic(fmt.Sprintf("embedded gameconfig.json parse failed: %v", err))
	}
	c, err := NewFromPB(t)
	if err != nil {
		panic(fmt.Sprintf("embedded battle config invalid: %v", err))
	}
	return c
}
