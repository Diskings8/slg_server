package gacha

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/common/utils/util_jsons"
)

// gachaJSON 抽卡配置表 JSON 结构（pools → drop_groups → items 多表嵌套）。
// 复用 RecruitPoolConfig/DropGroupConfig/DropItemConfig 作为表行（已带 json tag）。
type gachaJSON struct {
	Pools []*RecruitPoolConfig `json:"pools"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "gacha" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建抽卡配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j gachaJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	pools := make(map[int32]*RecruitPoolConfig, len(j.Pools))
	for _, p := range j.Pools {
		if p.PoolID <= 0 {
			return fmt.Errorf("pool_id must be > 0, got %d", p.PoolID)
		}
		if _, dup := pools[p.PoolID]; dup {
			return fmt.Errorf("duplicate pool_id %d", p.PoolID)
		}
		pools[p.PoolID] = p
	}

	c.pools = pools
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验抽卡配置完整性（主键唯一/消耗/掉落组/保底引用/心愿去重）
func (c *Conf) Validate() error {
	if len(c.pools) == 0 {
		return fmt.Errorf("pools must not be empty")
	}
	for poolID, pool := range c.pools {
		if poolID <= 0 {
			return fmt.Errorf("pool_id must be > 0, got %d", poolID)
		}
		// 消耗：非负且至少一种 > 0；有券模式必须单抽券 > 0
		if pool.SingleTicket < 0 || pool.SingleGold < 0 || pool.TenTicket < 0 || pool.TenGold < 0 {
			return fmt.Errorf("pool %d consume values must be >= 0", poolID)
		}
		if pool.SingleTicket+pool.SingleGold+pool.TenTicket+pool.TenGold == 0 {
			return fmt.Errorf("pool %d must have at least one consume type > 0", poolID)
		}
		if pool.TicketConfID != 0 && pool.SingleTicket <= 0 {
			return fmt.Errorf("pool %d ticket_conf_id set but single_ticket must be > 0", poolID)
		}
		// 掉落组：组ID 唯一、权重 > 0、条目非空且权重 > 0
		if len(pool.DropGroups) == 0 {
			return fmt.Errorf("pool %d drop_groups must not be empty", poolID)
		}
		groupIDs := make(map[int32]struct{}, len(pool.DropGroups))
		for _, g := range pool.DropGroups {
			if g.GroupID <= 0 {
				return fmt.Errorf("pool %d drop group_id must be > 0, got %d", poolID, g.GroupID)
			}
			if _, dup := groupIDs[g.GroupID]; dup {
				return fmt.Errorf("pool %d duplicate drop group_id %d", poolID, g.GroupID)
			}
			groupIDs[g.GroupID] = struct{}{}
			if g.Weight <= 0 {
				return fmt.Errorf("pool %d group %d weight must be > 0", poolID, g.GroupID)
			}
			if len(g.Items) == 0 {
				return fmt.Errorf("pool %d group %d items must not be empty", poolID, g.GroupID)
			}
			for _, it := range g.Items {
				if it.Weight <= 0 {
					return fmt.Errorf("pool %d group %d item weight must be > 0", poolID, g.GroupID)
				}
				if it.Count < 0 {
					return fmt.Errorf("pool %d group %d item count must be >= 0", poolID, g.GroupID)
				}
			}
		}
		// 保底/首抽组引用须在本池
		if pool.GuaranteeTimes < 0 {
			return fmt.Errorf("pool %d guarantee_times must be >= 0", poolID)
		}
		if pool.GuaranteeGroupID != 0 {
			if _, ok := groupIDs[pool.GuaranteeGroupID]; !ok {
				return fmt.Errorf("pool %d guarantee_group_id %d not found in drop_groups", poolID, pool.GuaranteeGroupID)
			}
		}
		if pool.FirstDropGroupID != 0 {
			if _, ok := groupIDs[pool.FirstDropGroupID]; !ok {
				return fmt.Errorf("pool %d first_drop_group_id %d not found in drop_groups", poolID, pool.FirstDropGroupID)
			}
		}
		// 心愿：元素 > 0 且去重
		seen := make(map[int32]struct{}, len(pool.WishHeros))
		for _, h := range pool.WishHeros {
			if h <= 0 {
				return fmt.Errorf("pool %d wish_heros element must be > 0", poolID)
			}
			if _, dup := seen[h]; dup {
				return fmt.Errorf("pool %d duplicate wish hero %d", poolID, h)
			}
			seen[h] = struct{}{}
		}
		if pool.WishTimes < 0 {
			return fmt.Errorf("pool %d wish_times must be >= 0", poolID)
		}
	}
	return nil
}
