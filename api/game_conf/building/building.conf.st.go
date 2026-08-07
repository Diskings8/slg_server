// Package building 玩家城市内建筑配置表（Go 内嵌占位数据，后续可迁 JSON）
//
// 城市内建筑：主城 + 校场/兵营/城墙/仓库 + 资源建筑（产出预留）。
// 分城（RoleBranchCity）归 worldmap OverlayEvent，不在本表。
// 建造/升级：消耗资源（BuildCost/UpgradeCostBase×growth）+ 建造时长（BuildTimeUx/UpgradeTimeGrowth），
// 惰性结算（Constructing + EndTimeUx，查询/登录时判断到期）。
package building

import (
	"math"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// LevelNum 等级→数值断点（稀疏，Load 时前向填充为稠密数组）
type LevelNum struct {
	Level uint32 `json:"level"`
	Num   uint32 `json:"num"`
}

// BuildingConf 单建筑类型配置
type BuildingConf struct {
	Type      pb_city.BuildingType      `json:"type"`
	Name      string                    `json:"name"`
	Footprint pb_city.BuildingFootprint `json:"footprint"`
	MaxLevel  uint32                    `json:"max_level"` // 等级上限

	BuildTimeUx int64 `json:"build_time_ux"` // 建造耗时（秒）；0=即时
	BuildCost   []common_declarations.ItemUse `json:"-"` // 建造消耗（JSON 经 costJSON 转换，本轮可空=免费）

	UpgradeCostBase   []common_declarations.ItemUse `json:"-"` // 1→2 升级消耗基数
	UpgradeCostGrowth float64                       `json:"upgrade_cost_growth"` // 升级消耗成长（每级乘数）
	UpgradeTimeGrowth float64                       `json:"upgrade_time_growth"` // 升级耗时成长（每级乘数）

	// ── 功能解锁参数（仅对特定类型生效，其余置零） ──
	QueueNums       []LevelNum      `json:"queue_nums"`         // 校场：等级→队列数断点
	DefensePerLevel uint32          `json:"defense_per_level"`  // 城墙：每级防御加成（预留）
	CapPerLevel     uint64          `json:"cap_per_level"`      // 仓库：每级资源存量上限（预留）
	ProduceItem     pb_confs.ItemID `json:"produce_item"`       // 资源建筑：产出资源配置ID（预留）
	ProducePerHourL int64           `json:"produce_per_hour_l"` // 资源建筑：每级每小时产出（预留）
}

// Conf 建筑配置聚合
type Conf struct {
	buildingByType map[pb_city.BuildingType]*BuildingConf
	queueNums      []uint32 // 校场等级→队列数稠密数组（index=等级）
	maxDrillLevel  uint32

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造建筑配置（内置占位数据）
func New() *Conf {
	c := &Conf{
		buildingByType: make(map[pb_city.BuildingType]*BuildingConf),
	}
	// 与 json/building.json 一致的内嵌占位
	c.buildingByType[pb_city.BuildingType_RoleMainCity] = &BuildingConf{
		Type: pb_city.BuildingType_RoleMainCity, Name: "main_city", Footprint: 9, MaxLevel: 10,
		BuildTimeUx: 300, UpgradeCostGrowth: 1.6, UpgradeTimeGrowth: 1.2,
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 500}},
	}
	c.buildingByType[pb_city.BuildingType_RoleBarracks] = &BuildingConf{
		Type: pb_city.BuildingType_RoleBarracks, Name: "barracks", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 120, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:      []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 200}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 300}},
	}
	c.buildingByType[pb_city.BuildingType_RoleDrill] = &BuildingConf{
		Type: pb_city.BuildingType_RoleDrill, Name: "drill", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 200}},
		QueueNums:       []LevelNum{{Level: 1, Num: 1}, {Level: 2, Num: 2}, {Level: 5, Num: 3}},
	}
	c.buildingByType[pb_city.BuildingType_RoleWall] = &BuildingConf{
		Type: pb_city.BuildingType_RoleWall, Name: "wall", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.4, UpgradeTimeGrowth: 1.2,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		DefensePerLevel: 100,
	}
	c.buildingByType[pb_city.BuildingType_RoleWarehouse] = &BuildingConf{
		Type: pb_city.BuildingType_RoleWarehouse, Name: "warehouse", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.4, UpgradeTimeGrowth: 1.2,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		CapPerLevel:     100000,
	}
	c.buildingByType[pb_city.BuildingType_RoleFarm] = &BuildingConf{
		Type: pb_city.BuildingType_RoleFarm, Name: "farm", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		ProduceItem:     100003, ProducePerHourL: 100,
	}
	c.buildingByType[pb_city.BuildingType_RoleLumber] = &BuildingConf{
		Type: pb_city.BuildingType_RoleLumber, Name: "lumber", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		ProduceItem:     100004, ProducePerHourL: 100,
	}
	c.buildingByType[pb_city.BuildingType_RoleStone] = &BuildingConf{
		Type: pb_city.BuildingType_RoleStone, Name: "stone", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		ProduceItem:     100005, ProducePerHourL: 100,
	}
	c.buildingByType[pb_city.BuildingType_RoleIron] = &BuildingConf{
		Type: pb_city.BuildingType_RoleIron, Name: "iron", Footprint: 4, MaxLevel: 10,
		BuildTimeUx: 60, UpgradeCostGrowth: 1.5, UpgradeTimeGrowth: 1.3,
		BuildCost:       []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 100}},
		UpgradeCostBase: []common_declarations.ItemUse{{ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: 150}},
		ProduceItem:     100006, ProducePerHourL: 100,
	}
	c.rebuildQueueNums()
	return c
}

// rebuildQueueNums 由校场配置构建队列数稠密数组
func (c *Conf) rebuildQueueNums() {
	drill := c.buildingByType[pb_city.BuildingType_RoleDrill]
	if drill == nil || len(drill.QueueNums) == 0 {
		c.queueNums = []uint32{0}
		c.maxDrillLevel = 0
		return
	}
	maxLevel := drill.QueueNums[len(drill.QueueNums)-1].Level
	nums := make([]uint32, maxLevel+1)
	prev := uint32(1) // 默认 1 队列
	for i := uint32(1); i <= maxLevel; i++ {
		for _, q := range drill.QueueNums {
			if q.Level == i {
				prev = q.Num
				break
			}
		}
		nums[i] = prev
	}
	c.queueNums = nums
	c.maxDrillLevel = maxLevel
}

// GetBuilding 按类型查询建筑配置
func (c *Conf) GetBuilding(t pb_city.BuildingType) (*BuildingConf, bool) {
	b, ok := c.buildingByType[t]
	return b, ok
}

// QueueNumAtLevel 校场等级对应队列数（无配置默认 1）
func (c *Conf) QueueNumAtLevel(level uint32) uint32 {
	if c.maxDrillLevel == 0 || level == 0 {
		return 1
	}
	if level > c.maxDrillLevel {
		return c.queueNums[c.maxDrillLevel]
	}
	return c.queueNums[level]
}

// UpgradeCost 升级 curLevel → curLevel+1 的消耗 = base × growth^(curLevel-1)，每项 ceil
func (c *Conf) UpgradeCost(t pb_city.BuildingType, curLevel uint32) ([]common_declarations.ItemUse, bool) {
	b, ok := c.buildingByType[t]
	if !ok || len(b.UpgradeCostBase) == 0 {
		return nil, ok
	}
	mult := math.Pow(b.UpgradeCostGrowth, float64(maxInt32(curLevel-1)))
	cost := make([]common_declarations.ItemUse, len(b.UpgradeCostBase))
	for i, base := range b.UpgradeCostBase {
		cost[i] = base
		cost[i].Count = int64(math.Ceil(float64(base.Count) * mult))
	}
	return cost, true
}

// UpgradeTime 升级 curLevel → curLevel+1 的耗时 = ceil(BuildTimeUx × growth^(curLevel-1))
func (c *Conf) UpgradeTime(t pb_city.BuildingType, curLevel uint32) int64 {
	b, ok := c.buildingByType[t]
	if !ok {
		return 0
	}
	mult := math.Pow(b.UpgradeTimeGrowth, float64(maxInt32(curLevel-1)))
	return int64(math.Ceil(float64(b.BuildTimeUx) * mult))
}

// AllTypes 全部建筑类型（供跨表校验/测试）
func (c *Conf) AllTypes() []pb_city.BuildingType {
	types := make([]pb_city.BuildingType, 0, len(c.buildingByType))
	for t := range c.buildingByType {
		types = append(types, t)
	}
	return types
}

func maxInt32(v uint32) int32 {
	if v > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v)
}
