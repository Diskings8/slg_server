package gacha

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// groupKey 掉落条目归属（池+组）。
type groupKey struct{ pool, group int32 }

// NewFromPB 从 tabtoy 单一配置构建抽卡配置（gacha_pool + gacha_drop_group + gacha_drop_item 三表拼装）。
//
// 迁移原 Load 的「局部构建 + 末尾提交 + 校验」：构造失败返回 err，不产生半更新。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	// 掉落条目按（池,组）聚合，保持导出顺序（gacha_drop_item 行序 = 组内条目序）
	itemsByGroup := make(map[groupKey][]DropItemConfig, len(t.GachaDropItem))
	for _, it := range t.GachaDropItem {
		k := groupKey{it.PoolId, it.GroupId}
		itemsByGroup[k] = append(itemsByGroup[k], DropItemConfig{
			RewardConfID:   it.RewardConfId,
			IsHero:         it.IsHero,
			Count:          it.Count,
			Weight:         it.Weight,
			GuaranteeReset: it.GuaranteeReset,
		})
	}

	pools := make(map[int32]*RecruitPoolConfig, len(t.GachaPool))
	for _, p := range t.GachaPool {
		if p.PoolId <= 0 {
			return nil, fmt.Errorf("pool_id must be > 0, got %d", p.PoolId)
		}
		if _, dup := pools[p.PoolId]; dup {
			return nil, fmt.Errorf("duplicate pool_id %d", p.PoolId)
		}
		pool := &RecruitPoolConfig{
			PoolID:           p.PoolId,
			Name:             p.Name,
			TicketConfID:     p.TicketConfId,
			SingleTicket:     p.SingleTicket,
			SingleGold:       p.SingleGold,
			TenTicket:        p.TenTicket,
			TenGold:          p.TenGold,
			FreeDaily:        p.FreeDaily,
			HalfPrice:        p.HalfPrice,
			GuaranteeTimes:   p.GuaranteeTimes,
			GuaranteeGroupID: p.GuaranteeGroupId,
			FirstDropGroupID: p.FirstDropGroupId,
			WishHeros:        p.WishHeros,
			WishTimes:        p.WishTimes,
		}
		for _, g := range t.GachaDropGroup {
			if g.PoolId != p.PoolId {
				continue
			}
			pool.DropGroups = append(pool.DropGroups, DropGroupConfig{
				GroupID: g.GroupId,
				Weight:  g.Weight,
				Items:   itemsByGroup[groupKey{p.PoolId, g.GroupId}],
			})
		}
		if len(pool.DropGroups) == 0 {
			return nil, fmt.Errorf("pool %d drop_groups must not be empty", p.PoolId)
		}
		pools[p.PoolId] = pool
	}

	c := &Conf{pools: pools}
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("gacha validate: %w", err)
	}
	return c, nil
}

// New 构造抽卡配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
func New() *Conf {
	c, err := gameconfigjson.Build(NewFromPB)
	if err != nil {
		panic(fmt.Sprintf("embedded config invalid: %v", err))
	}
	return c
}
