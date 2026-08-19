package building

import (
	"fmt"

	gameconfigjson "server.slg.com/api/game_conf/json"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_gameconfig"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// NewFromPB 从 tabtoy 单一配置构建建筑配置（building 表逐行 → buildingByType 索引）。
//
// 迁移原 Load 的「局部构建 + 末尾提交 + 校验」：构造失败返回 err，不产生半更新。
func NewFromPB(t *pb_gameconfig.Table) (*Conf, error) {
	byType := make(map[pb_city.BuildingType]*BuildingConf, len(t.Building))
	for _, row := range t.Building {
		tp := pb_city.BuildingType(row.Btype)
		if _, dup := byType[tp]; dup {
			return nil, fmt.Errorf("duplicate building type %d", tp)
		}
		byType[tp] = &BuildingConf{
			Type:              tp,
			Name:              row.Name,
			Footprint:         pb_city.BuildingFootprint(row.Footprint),
			MaxLevel:          row.MaxLevel,
			BuildTimeUx:       row.BuildTimeUx,
			BuildCost:         pbCosts(row.BuildCost),
			UpgradeCostBase:   pbCosts(row.UpgradeCostBase),
			UpgradeCostGrowth: float64(row.UpgradeCostGrowth),
			UpgradeTimeGrowth: float64(row.UpgradeTimeGrowth),
			QueueNums:         pbLevelNums(row.QueueNums),
			DefensePerLevel:   row.DefensePerLevel,
			CapPerLevel:       row.CapPerLevel,
			ProduceItem:       pb_confs.ItemID(row.ProduceItem),
			ProducePerHourL:   row.ProducePerHourL,
		}
	}
	c := &Conf{buildingByType: byType}
	c.rebuildQueueNums()
	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("building validate: %w", err)
	}
	return c, nil
}

// New 构造建筑配置（内嵌 gameconfig.json 兜底；数据损坏时 panic，正常不应发生）。
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

// pbLevelNums 将 pb level_num 列表转换为领域 LevelNum（校场队列数断点）。
func pbLevelNums(rows []*pb_gameconfig.LevelNum) []LevelNum {
	if len(rows) == 0 {
		return nil
	}
	out := make([]LevelNum, 0, len(rows))
	for _, r := range rows {
		out = append(out, LevelNum{Level: r.Level, Num: r.Num})
	}
	return out
}
