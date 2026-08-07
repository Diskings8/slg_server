package building

import (
	"fmt"

	"server.slg.com/api/game_conf/table"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
)

// itemUseJSON 域内 ItemUse 的 JSON 镜像（common_declarations.ItemUse 无 json tag，需中间结构转换）
type itemUseJSON struct {
	ItemID   int32 `json:"item_id"`
	ItemType int32 `json:"item_type,omitempty"`
	Count    int64 `json:"count"`
}

func (j itemUseJSON) toItemUse() common_declarations.ItemUse {
	return common_declarations.ItemUse{
		ItemID:   pb_confs.ItemID(j.ItemID),
		ItemType: pb_confs.ItemType(j.ItemType),
		Count:    j.Count,
	}
}

func toCosts(rows []itemUseJSON) []common_declarations.ItemUse {
	if len(rows) == 0 {
		return nil
	}
	out := make([]common_declarations.ItemUse, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.toItemUse())
	}
	return out
}

// buildingRowJSON 单建筑类型配置行（磁盘格式，snake_case）
type buildingRowJSON struct {
	Type      pb_city.BuildingType      `json:"type"`
	Name      string                    `json:"name"`
	Footprint pb_city.BuildingFootprint `json:"footprint"`
	MaxLevel  uint32                    `json:"max_level"`

	BuildTimeUx int64        `json:"build_time_ux"`
	BuildCost   []itemUseJSON `json:"build_cost"`

	UpgradeCostBase   []itemUseJSON `json:"upgrade_cost_base"`
	UpgradeCostGrowth float64       `json:"upgrade_cost_growth"`
	UpgradeTimeGrowth float64       `json:"upgrade_time_growth"`

	QueueNums       []LevelNum      `json:"queue_nums"`
	DefensePerLevel uint32          `json:"defense_per_level"`
	CapPerLevel     uint64          `json:"cap_per_level"`
	ProduceItem     pb_confs.ItemID `json:"produce_item"`
	ProducePerHourL int64           `json:"produce_per_hour_l"`
}

// buildingJSON 建筑配置表 JSON 结构
type buildingJSON struct {
	Buildings []buildingRowJSON `json:"buildings"`
}

// FileName 表名（JSON 文件名，不含扩展名）
func (c *Conf) FileName() string { return "building" }

// Version 内容版本（JSON 加载后为内容 hash；Go 内嵌为 ""）
func (c *Conf) Version() string { return c.version }

// Load 从 JSON 字节构建建筑配置（覆盖占位）。失败保持本表数据不变（局部构建 + 末尾提交）。
func (c *Conf) Load(data []byte) error {
	var j buildingJSON
	if err := util_jsons.Unmarshal(data, &j); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	byType := make(map[pb_city.BuildingType]*BuildingConf, len(j.Buildings))
	for _, row := range j.Buildings {
		if _, dup := byType[row.Type]; dup {
			return fmt.Errorf("duplicate building type %d", row.Type)
		}
		byType[row.Type] = &BuildingConf{
			Type:              row.Type,
			Name:              row.Name,
			Footprint:         row.Footprint,
			MaxLevel:          row.MaxLevel,
			BuildTimeUx:       row.BuildTimeUx,
			BuildCost:         toCosts(row.BuildCost),
			UpgradeCostBase:   toCosts(row.UpgradeCostBase),
			UpgradeCostGrowth: row.UpgradeCostGrowth,
			UpgradeTimeGrowth: row.UpgradeTimeGrowth,
			QueueNums:         row.QueueNums,
			DefensePerLevel:   row.DefensePerLevel,
			CapPerLevel:       row.CapPerLevel,
			ProduceItem:       row.ProduceItem,
			ProducePerHourL:   row.ProducePerHourL,
		}
	}

	// 末尾一次性提交（局部构建 + 校验通过后 swap）
	c.buildingByType = byType
	c.rebuildQueueNums()
	c.version = table.ContentHash(data)
	return nil
}

// Validate 校验建筑配置完整性
func (c *Conf) Validate() error {
	if len(c.buildingByType) == 0 {
		return fmt.Errorf("buildings must not be empty")
	}
	for t, b := range c.buildingByType {
		if t <= 0 {
			return fmt.Errorf("building type must be > 0, got %d", t)
		}
		if b.Footprint != 4 && b.Footprint != 9 {
			return fmt.Errorf("building %d footprint must be 4 or 9, got %d", t, b.Footprint)
		}
		if b.MaxLevel == 0 {
			return fmt.Errorf("building %d max_level must be > 0", t)
		}
		if b.BuildTimeUx < 0 {
			return fmt.Errorf("building %d build_time_ux must be >= 0", t)
		}
		if b.UpgradeCostGrowth <= 0 || b.UpgradeTimeGrowth <= 0 {
			return fmt.Errorf("building %d upgrade growth must be > 0", t)
		}
		for _, cost := range b.BuildCost {
			if cost.Count < 0 {
				return fmt.Errorf("building %d build_cost count must be >= 0", t)
			}
		}
		for _, cost := range b.UpgradeCostBase {
			if cost.Count < 0 {
				return fmt.Errorf("building %d upgrade_cost_base count must be >= 0", t)
			}
		}
		// 校场：queue_nums 非空、level 升序、首项 level==1
		if t == pb_city.BuildingType_RoleDrill {
			if len(b.QueueNums) == 0 {
				return fmt.Errorf("drill queue_nums must not be empty")
			}
			if b.QueueNums[0].Level != 1 {
				return fmt.Errorf("drill queue_nums first level must be 1, got %d", b.QueueNums[0].Level)
			}
			for i := 1; i < len(b.QueueNums); i++ {
				if b.QueueNums[i].Level <= b.QueueNums[i-1].Level {
					return fmt.Errorf("drill queue_nums level must be ascending, got %d after %d",
						b.QueueNums[i].Level, b.QueueNums[i-1].Level)
				}
			}
		}
		if b.DefensePerLevel < 0 || b.CapPerLevel < 0 || b.ProducePerHourL < 0 {
			return fmt.Errorf("building %d unlock params must be >= 0", t)
		}
	}
	return nil
}
