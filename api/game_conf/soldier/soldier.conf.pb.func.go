package soldier

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// NewFromPB 从 tabtoy 单一配置构建兵力配置（soldier 单行 + soldier_hero_cap/soldier_barrack_bonus 断点）。
//
// 与 Load 同构：断点前向填充为稠密数组，运行期 O(1) 查询。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	if len(t.Soldier) == 0 {
		return nil, fmt.Errorf("soldier table empty")
	}

	heroRows := make([]heroLevelCap, 0, len(t.SoldierHeroCap))
	for _, r := range t.SoldierHeroCap {
		heroRows = append(heroRows, heroLevelCap{Level: r.Level, SoldierNum: r.SoldierNum})
	}
	barrackRows := make([]barrackLevelBonus, 0, len(t.SoldierBarrackBonus))
	for _, r := range t.SoldierBarrackBonus {
		barrackRows = append(barrackRows, barrackLevelBonus{Level: r.Level, Bonus: r.Bonus})
	}

	maxHero := maxLevel(heroRows)
	maxBarrack := maxLevelBonus(barrackRows)
	c := &Conf{
		DefaultSoldierNum: t.Soldier[0].DefaultSoldierNum,
		heroCaps:          make([]uint32, maxHero+1),
		barrackBonus:      make([]uint32, maxBarrack+1),
		maxHeroLevel:      int32(maxHero),
		maxBarrackLevel:   int32(maxBarrack),
	}
	fillHeroCaps(c.heroCaps, heroRows)
	fillBarrackBonus(c.barrackBonus, barrackRows)

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("soldier validate: %w", err)
	}
	return c, nil
}

// New 构造兵力配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
