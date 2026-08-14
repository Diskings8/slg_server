package worldmap_inits

import (
	"math/rand/v2"

	common_configs "server.slg.com/common/configs"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
)

// defaultMapSeed 默认地图生成种子（未配置 worldmap.seed 时回落）
const defaultMapSeed = int64(20260731)

// resolveMapSeed 读取地图生成种子：优先配置（worldmap.seed），未配置/非正回落默认值。
// 同种子 → 同底图，重启后地图稳定，DB 动态状态才能作为覆盖层正确叠加。
func resolveMapSeed() int64 {
	if cfg := common_configs.GetConf(); cfg != nil && cfg.Worldmap.Seed > 0 {
		return cfg.Worldmap.Seed
	}
	return defaultMapSeed
}

// MapElementConf 地形元素配置
type MapElementConf struct {
	ElementType cores_declarations.ElementType
	ConfigID    uint32                      // 元素配置ID
	Level       cores_declarations.MapLevel // 元素等级
	Weight      int                         // 生成权重（相对权重，越大出现越多）
	CanBorn     bool                        // 是否可作出生点（IsCantBornUse 为准）
}

// ResourceLevelConf 资源等级分布配置
type ResourceLevelConf struct {
	Level    cores_declarations.MapLevel // 资源等级 1~9
	Weight   int                         // 相对权重（越大越常见）
	MaxCount int                         // 全图可分配数量上限；0=不设上限（lv1 兜底）
}

// MapGenerateConfig 地图生成配置
type MapGenerateConfig struct {
	Seed           int64
	Terrain        []MapElementConf    // 地形元素（按权重）
	ResourceLevels []ResourceLevelConf // 资源等级分布（降序 lv9..lv1）
	ResourceTypes  int                 // 资源类型数（Resources_1..N，均匀分布）
}

// defaultTerrainElements 默认地形元素（可诞生，占 80%）
var defaultTerrainElements = []MapElementConf{
	{ElementType: cores_declarations.ElementType_Terrain_1, Weight: 45, CanBorn: true}, // 平原
	{ElementType: cores_declarations.ElementType_Terrain_2, Weight: 25, CanBorn: true}, // 丘陵/草地
	{ElementType: cores_declarations.ElementType_Terrain_3, Weight: 10, CanBorn: true}, // 战乱地
}

// defaultResourceLevels 默认资源等级分布（降序排列；权重越高越常见，数量上限约束稀有等级）。
//
// 平衡思路：
//   - lv5 是【关键分水岭】——产量为 lv4 的 3 倍，核心玩法主占 lv5 地 → 权重最高、上限最大
//   - lv1~5 为主力占有带（权重高、上限大）
//   - lv6~9 高端地稀少（产量仅逐级 +200）→ 权重低、上限小
//   - lv1 兜底（MaxCount=0 不设上限）：较高等级达上限后降级，最终兜底 lv1
var defaultResourceLevels = []ResourceLevelConf{
	{Level: 9, Weight: 1, MaxCount: 100},
	{Level: 8, Weight: 3, MaxCount: 300},
	{Level: 7, Weight: 5, MaxCount: 600},
	{Level: 6, Weight: 8, MaxCount: 1000},
	{Level: 5, Weight: 22, MaxCount: 50000}, // 关键分水岭，主占用地
	{Level: 4, Weight: 18, MaxCount: 40000},
	{Level: 3, Weight: 14, MaxCount: 35000},
	{Level: 2, Weight: 10, MaxCount: 30000},
	{Level: 1, Weight: 8, MaxCount: 0}, // 兜底
}

// defaultResourceTypeCount 默认资源类型数（Resources_1~4）
const defaultResourceTypeCount = 4

// NewDefaultGenerateConfig 构建默认生成配置
func NewDefaultGenerateConfig(seed int64) *MapGenerateConfig {
	terrain := make([]MapElementConf, len(defaultTerrainElements))
	copy(terrain, defaultTerrainElements)
	levels := make([]ResourceLevelConf, len(defaultResourceLevels))
	copy(levels, defaultResourceLevels)
	return &MapGenerateConfig{
		Seed:           seed,
		Terrain:        terrain,
		ResourceLevels: levels,
		ResourceTypes:  defaultResourceTypeCount,
	}
}

// InitMapElements 初始化地图元素 — 限定地形元素集合 + 种子确定性生成
func InitMapElements(mdm *map_datas.MapDataManager, seed int64) {
	cfg := NewDefaultGenerateConfig(seed)
	mdm.GenerateMap(cfg.Generator())
}

// Generator 返回按权重随机生成元素类型的生成函数
//
// 生成规则：
//   - 80% 地形（Terrain_1/2/3 按权重）
//   - 20% 资源：类型均匀（Resources_1..N）→ 等级按权重选 lv1~lv9，达上限则降级（lv1 兜底）
//
// 确定性保证：同一种子 + 固定遍历顺序（0..MapCount-1）→ 生成完全一致的地图。
// 等级分配计数随遍历顺序确定性推进，同一级满后降级路径一致。
func (c *MapGenerateConfig) Generator() func(mapID cores_declarations.MapID, x, y int32) (cores_declarations.ElementType, uint32, cores_declarations.MapLevel) {
	// 确定性 PRNG：PCG（Go 1.22+ rand/v2），两路种子混合
	rng := rand.New(rand.NewPCG(uint64(c.Seed), uint64(c.Seed)<<32|0x9e3779b97f4a7c15))
	// 资源等级已分配数量（index=等级 1~9）
	counts := make([]int, 10)

	terrainTotal := 0
	for _, e := range c.Terrain {
		terrainTotal += e.Weight
	}
	if terrainTotal <= 0 {
		terrainTotal = 1
	}
	levelTotal := 0
	for _, lc := range c.ResourceLevels {
		levelTotal += lc.Weight
	}
	if levelTotal <= 0 {
		levelTotal = 1
	}
	if c.ResourceTypes <= 0 {
		c.ResourceTypes = 1
	}

	return func(_ cores_declarations.MapID, _, _ int32) (cores_declarations.ElementType, uint32, cores_declarations.MapLevel) {
		// 20% 资源
		if rng.IntN(100) >= 80 {
			typeIdx := rng.IntN(c.ResourceTypes)
			level := c.pickResourceLevel(rng, levelTotal, counts)
			et := cores_declarations.ElementType(int(cores_declarations.ElementType_Resources_1) + typeIdx)
			return et, uint32(1001 + typeIdx), level
		}
		// 80% 地形：按权重
		n := rng.IntN(terrainTotal)
		for _, e := range c.Terrain {
			n -= e.Weight
			if n < 0 {
				return e.ElementType, e.ConfigID, e.Level
			}
		}
		return cores_declarations.ElementType_Terrain_1, 0, 0
	}
}

// pickResourceLevel 按权重选资源等级；选中等级若已达上限则降级（lv1 兜底）
func (c *MapGenerateConfig) pickResourceLevel(rng *rand.Rand, levelTotal int, counts []int) cores_declarations.MapLevel {
	n := rng.IntN(levelTotal)
	for _, lc := range c.ResourceLevels {
		n -= lc.Weight
		if n < 0 {
			return c.degradeLevel(lc.Level, counts)
		}
	}
	counts[1]++
	return 1
}

// degradeLevel 从选中等级向下找第一个未达上限的等级；全部满则 lv1 兜底。
// ResourceLevels 按降序排列，天然从高到低尝试。
func (c *MapGenerateConfig) degradeLevel(level cores_declarations.MapLevel, counts []int) cores_declarations.MapLevel {
	for _, lc := range c.ResourceLevels {
		if lc.Level > level {
			continue
		}
		if lc.MaxCount == 0 || counts[lc.Level] < lc.MaxCount {
			counts[lc.Level]++
			return lc.Level
		}
	}
	counts[1]++
	return 1
}
