// map_data_gen — 地图模拟数据生成器（自包含）
//
// 与 services/worldmap 服务端完全一致的确定性生成：
//   - 同一种子 (20260731) + PCG PRNG + 默认权重元素集合（复制自 worldmap_inits/map.generate.func.go）
//   - 从 mapID=0 完整走一遍全图序列（与服务器 GenerateMap 一致），只记录目标区域格子
//
// 输出：MapCellInfo 兼容的 JSON（含坐标与元素名，便于阅读/前端模拟）
package main

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
)

const (
	seed  = int64(20260731) // 服务端 defaultMapSeed
	scope = int32(1000)     // 服务端 DefaultMapConfig.MapScope

	// 目标区域：以 (centerX, centerY) 为中心的 (half*2+1)×(half*2+1) 矩形
	centerX, centerY = int32(500), int32(500)
	half             = int32(50) // 每边 50 → 101×101
)

// element 复制自服务端 defaultMapElements（权重总和 100）
type element struct {
	et     int32
	config uint32
	level  int32
	weight int
}

var elements = []element{
	{5, 0, 0, 45}, // Terrain_1 山/平原（可出生）
	{6, 0, 0, 25}, // Terrain_2 水/丘陵（可出生）
	{7, 0, 0, 10}, // Terrain_3 战乱地（可出生）
	{1, 1001, 1, 8},
	{2, 1002, 1, 6},
	{3, 1003, 1, 4},
	{4, 1004, 1, 2},
}

var elementNames = map[int32]string{
	0: "None",
	1: "Resources_1",
	2: "Resources_2",
	3: "Resources_3",
	4: "Resources_4",
	5: "Terrain_1",
	6: "Terrain_2",
	7: "Terrain_3",
}

type cell struct {
	MapID       int32  `json:"map_id"`
	X           int32  `json:"x"`
	Y           int32  `json:"y"`
	ElementType int32  `json:"element_type"`
	ElementName string `json:"element_name"`
	ElementID   uint32 `json:"element_id"`
	Level       int32  `json:"level"`
	OwnerID     uint64 `json:"owner_id"`
	ServerID    uint32 `json:"server_id"`
}

type output struct {
	Meta struct {
		Seed      int64 `json:"seed"`
		Scope     int32 `json:"scope"`
		CenterX   int32 `json:"center_x"`
		CenterY   int32 `json:"center_y"`
		Width     int32 `json:"width"`
		Height    int32 `json:"height"`
		CellCount int   `json:"cell_count"`
	} `json:"meta"`
	Cells []cell `json:"cells"`
}

func main() {
	totalWeight := 0
	for _, e := range elements {
		totalWeight += e.weight
	}

	// 与服务端 Generator() 相同的 PCG 初始化
	rng := rand.New(rand.NewPCG(uint64(seed), uint64(seed)<<32|0x9e3779b97f4a7c15))
	gen := func() (et int32, cid uint32, lv int32) {
		n := rng.IntN(totalWeight)
		for _, e := range elements {
			n -= e.weight
			if n < 0 {
				return e.et, e.config, e.level
			}
		}
		return 5, 0, 0 // 兜底 Terrain_1
	}

	minX, maxX := centerX-half, centerX+half
	minY, maxY := centerY-half, centerY+half

	var cells []cell
	// 确定性序列：与服务器 GenerateMap 一致，mapID 递增完整遍历，跳过目标区外格子
	for id := int32(0); id < scope*scope; id++ {
		x, y := id%scope, id/scope
		et, cid, lv := gen()
		if x < minX || x > maxX || y < minY || y > maxY {
			continue
		}
		cells = append(cells, cell{
			MapID:       id,
			X:           x,
			Y:           y,
			ElementType: et,
			ElementName: elementNames[et],
			ElementID:   cid,
			Level:       lv,
		})
	}

	var out output
	out.Meta.Seed = seed
	out.Meta.Scope = scope
	out.Meta.CenterX = centerX
	out.Meta.CenterY = centerY
	out.Meta.Width = maxX - minX + 1
	out.Meta.Height = maxY - minY + 1
	out.Meta.CellCount = len(cells)
	out.Cells = cells

	terrain := map[string]int{}
	for _, c := range cells {
		terrain[c.ElementName]++
	}
	fmt.Printf("生成 %d 格 (x∈[%d,%d], y∈[%d,%d])\n", len(cells), minX, maxX, minY, maxY)
	fmt.Printf("分布: %v\n", terrain)

	outDir := "api/data_json"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		panic(err)
	}
	path := fmt.Sprintf("%s/map_data_%dx%d_%d_%d.json", outDir, out.Meta.Width, out.Meta.Height, centerX, centerY)
	buf, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("已写出: %s (%d bytes)\n", path, len(buf))
}
