package map_managers

import (
	"math"
	"sort"
	"testing"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/marchs"
)

// testMapConfig 测试用地图配置：1000×1000（本地实现 MapConfigI，避免依赖 worldmap）
type testMapConfig struct{}

func (testMapConfig) MapCount() int32 { return 1000 * 1000 }
func (testMapConfig) MapScope() int32 { return 1000 }
func (testMapConfig) MapID2XY(id cores_declarations.MapID) (x, y int32) {
	return id.Int32() % 1000, id.Int32() / 1000
}
func (testMapConfig) XY2MapID(x, y int32) cores_declarations.MapID {
	return cores_declarations.MapID(y*1000 + x)
}
func (testMapConfig) SortByDis(mapID cores_declarations.MapID, mapIDs []cores_declarations.MapID) {
	cx, cy := testMapConfig{}.MapID2XY(mapID)
	sort.Slice(mapIDs, func(i, j int) bool {
		ix, iy := testMapConfig{}.MapID2XY(mapIDs[i])
		jx, jy := testMapConfig{}.MapID2XY(mapIDs[j])
		di := (ix-cx)*(ix-cx) + (iy-cy)*(iy-cy)
		dj := (jx-cx)*(jx-cx) + (jy-cy)*(jy-cy)
		return di < dj
	})
}
func (testMapConfig) CoverMapIDs(id int32, _ int, i2 any) []cores_declarations.MapID {
	var r int32
	switch v := i2.(type) {
	case uint32:
		r = int32(v)
	case int:
		r = int32(v)
	default:
		r = 0
	}
	x := id % 1000
	y := id / 1000
	minX := int32(math.Max(float64(x-r), 0))
	maxX := int32(math.Min(float64(x+r), 999))
	minY := int32(math.Max(float64(y-r), 0))
	maxY := int32(math.Min(float64(y+r), 999))
	var ids []cores_declarations.MapID
	for ty := minY; ty <= maxY; ty++ {
		for tx := minX; tx <= maxX; tx++ {
			ids = append(ids, cores_declarations.MapID(ty*1000+tx))
		}
	}
	return ids
}

// setBornSafe 把区块 1 的候选种子区域（x/y∈[40,160]）设为可出生地形。
// 覆盖 initBornManager 的全部偏移种子（50/100/150）及其 3×3 邻域。
func setBornSafe(mdm *map_datas.MapDataManager) {
	for y := int32(40); y <= 160; y++ {
		for x := int32(40); x <= 160; x++ {
			mid := testMapConfig{}.XY2MapID(x, y)
			if info, ok := mdm.GetMapInfo(mid); ok {
				info.SetElement(cores_declarations.ElementType_Terrain_1, 0, 0)
			}
		}
	}
}

// TestNewMapManager_WiresBornBlockManager 验证 NewMapManager 完成出生块接线：
// mdm.BornBlockManager 非 nil，且 GetFreeBorn 能正常分配出生点。
func TestNewMapManager_WiresBornBlockManager(t *testing.T) {
	mdm := map_datas.NewMapDataManager(testMapConfig{}, "map_data")
	setBornSafe(mdm)

	tickerChan := make(chan *marchs.MarchInfo, 1000)
	marchMgr := marchs.New(tickerChan, "march_info", testMapConfig{}, cores_declarations.MarchTimeTypeStraight)
	_ = NewMapManager(1, cores_declarations.MapGroupBase, mdm, marchMgr, nil, nil)

	if mdm.BornBlockManager == nil {
		t.Fatal("BornBlockManager not wired by NewMapManager")
	}

	mapIDs, ls, bornID, _, _, err := mdm.GetFreeBorn()
	if err != nil {
		t.Fatalf("GetFreeBorn err: %v", err)
	}
	if len(mapIDs) != cores_declarations.HallCoverCount {
		t.Fatalf("got %d cells, want %d", len(mapIDs), cores_declarations.HallCoverCount)
	}
	if bornID < 1 {
		t.Fatalf("bornID = %d, want >= 1", bornID)
	}
	ls.Unlock()
}
