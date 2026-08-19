package hero

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// NewFromPB 从 tabtoy 单一配置构建英雄配置（hero 单行 + hero_exp 经验表 + hero_attr 属性摊平）。
//
// 迁移原 Load 的「局部构建 + 末尾提交 + 校验」：构造失败返回 err，不产生半更新。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Hero) == 0 {
		return nil, fmt.Errorf("hero table empty")
	}
	h := t.Hero[0]
	c := &Conf{
		MaxLevel:        h.MaxLevel,
		FreePointPer10L: h.FreePointPer_10L,
		MaxStarStage:    h.MaxStarStage,
		StarPointPer:    h.StarPointPer,
		AwakenLevel:     h.AwakenLevel,
		AwakenCost:      pbCosts(h.AwakenCost),
	}

	// 经验表：hero_exp 逐级（level=1..N）→ ExpNeed[index=level-1]
	expNeed := make([]uint32, len(t.HeroExp))
	for _, row := range t.HeroExp {
		if row.Level == 0 || int(row.Level) > len(expNeed) {
			return nil, fmt.Errorf("hero_exp level %d out of range [1,%d]", row.Level, len(expNeed))
		}
		expNeed[row.Level-1] = row.Exp
	}
	c.ExpNeed = expNeed

	// 英雄属性：hero_attr 摊平 base/growth，主键重复直接报错（不做静默覆盖）
	heroes := make(map[int32]HeroConf, len(t.HeroAttr))
	for _, row := range t.HeroAttr {
		if _, dup := heroes[row.ConfId]; dup {
			return nil, fmt.Errorf("duplicate hero conf_id %d", row.ConfId)
		}
		heroes[row.ConfId] = HeroConf{
			ConfID: row.ConfId,
			Base: HeroAttr{
				Attack: row.BaseAttack, Defense: row.BaseDefense, Intelligence: row.BaseIntelligence,
				Movement: row.BaseMovement, Relocation: row.BaseRelocation,
			},
			Growth: HeroAttr{
				Attack: row.GrowthAttack, Defense: row.GrowthDefense, Intelligence: row.GrowthIntelligence,
				Movement: row.GrowthMovement, Relocation: row.GrowthRelocation,
			},
			AttackRange: row.AttackRange,
		}
	}
	c.heroes = heroes

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("hero validate: %w", err)
	}
	return c, nil
}

// New 构造英雄配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}

// pbCosts 将 pb cost 列表转换为领域 ItemUse（与 JSON 路径 toCosts 同构）。
func pbCosts(costs []*pb_gameconfig.Cost) []common_declarations.ItemUse {
	if len(costs) == 0 {
		return nil
	}
	out := make([]common_declarations.ItemUse, 0, len(costs))
	for _, c := range costs {
		out = append(out, common_declarations.ItemUse{
			ItemID:   pb_confs.ItemID(c.ItemId),
			ItemType: pb_confs.ItemType(c.ItemType),
			Count:    c.Count,
		})
	}
	return out
}
