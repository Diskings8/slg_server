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

// MapElementConf 地图元素配置（限定地形元素集合的单个元素）
type MapElementConf struct {
	ElementType cores_declarations.ElementType
	ConfigID    uint32                       // 元素配置ID（资源/怪等）
	Level       cores_declarations.MapLevel  // 元素等级
	Weight      int                          // 生成权重（相对权重，越大出现越多）
	CanBorn     bool                         // 是否可作出生点（IsCantBornUse 为准）
}

// MapGenerateConfig 地图生成配置
type MapGenerateConfig struct {
	Seed     int64            // 生成种子：同种子 → 同地图
	Elements []MapElementConf // 限定地形元素集合
}

// defaultMapElements 默认限定地形元素集合
//
// TODO: 数据来源（配置表）确定后替换；当前为程序化占位分布。
// 权重总和 100，可诞生元素（Terrain_1/2/3）占 80%，保证出生点可用。
var defaultMapElements = []MapElementConf{
	{ElementType: cores_declarations.ElementType_Terrain_1, Weight: 45, CanBorn: true}, // 平原
	{ElementType: cores_declarations.ElementType_Terrain_2, Weight: 25, CanBorn: true}, // 丘陵/草地
	{ElementType: cores_declarations.ElementType_Terrain_3, Weight: 10, CanBorn: true}, // 战乱地
	{ElementType: cores_declarations.ElementType_Resources_1, ConfigID: 1001, Level: 1, Weight: 8}, // 资源1
	{ElementType: cores_declarations.ElementType_Resources_2, ConfigID: 1002, Level: 1, Weight: 6}, // 资源2
	{ElementType: cores_declarations.ElementType_Resources_3, ConfigID: 1003, Level: 1, Weight: 4}, // 资源3
	{ElementType: cores_declarations.ElementType_Resources_4, ConfigID: 1004, Level: 1, Weight: 2}, // 资源4
}

// NewDefaultGenerateConfig 构建默认生成配置
func NewDefaultGenerateConfig(seed int64) *MapGenerateConfig {
	elements := make([]MapElementConf, len(defaultMapElements))
	copy(elements, defaultMapElements)
	return &MapGenerateConfig{
		Seed:     seed,
		Elements: elements,
	}
}

// InitMapElements 初始化地图元素 — 限定元素集合 + 种子确定性生成
func InitMapElements(mdm *map_datas.MapDataManager, seed int64) {
	cfg := NewDefaultGenerateConfig(seed)
	mdm.GenerateMap(cfg.Generator())
}

// Generator 返回按权重随机生成元素类型的生成函数
//
// 确定性保证：同一种子 + 固定遍历顺序（0..MapCount-1）→ 生成完全一致的地图。
// 生成顺序由 MapDataManager.GenerateMap 的递增 mapID 循环保证。
func (c *MapGenerateConfig) Generator() func(mapID cores_declarations.MapID, x, y int32) (cores_declarations.ElementType, uint32, cores_declarations.MapLevel) {
	// 确定性 PRNG：PCG（Go 1.22+ rand/v2），两路种子混合
	rng := rand.New(rand.NewPCG(uint64(c.Seed), uint64(c.Seed)<<32|0x9e3779b97f4a7c15))

	totalWeight := 0
	for _, e := range c.Elements {
		totalWeight += e.Weight
	}
	if totalWeight <= 0 {
		totalWeight = 1
	}

	return func(_ cores_declarations.MapID, _, _ int32) (cores_declarations.ElementType, uint32, cores_declarations.MapLevel) {
		n := rng.IntN(totalWeight)
		for _, e := range c.Elements {
			n -= e.Weight
			if n < 0 {
				return e.ElementType, e.ConfigID, e.Level
			}
		}
		// 兜底：可诞生地形
		return cores_declarations.ElementType_Terrain_1, 0, 0
	}
}
