package worldmap_inits

import (
	"encoding/json"
	"os"
	"testing"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas"
)

// newTestMapData 构建完整的地图数据链路：
// NewDefaultMapConfig → NewMapDataManager → InitMapElements（程序化生成）
func newTestMapData(t *testing.T) *map_datas.MapDataManager {
	t.Helper()
	mdm := map_datas.NewMapDataManager(NewDefaultMapConfig(), "map_data")
	InitMapElements(mdm, defaultMapSeed)
	return mdm
}

// 1. 地图配置：坐标 ↔ MapID 双向换算
func TestDefaultMapConfig_XYRoundTrip(t *testing.T) {
	cfg := NewDefaultMapConfig()
	if cfg.MapCount() != 1000*1000 {
		t.Fatalf("MapCount = %d, want %d", cfg.MapCount(), 1000*1000)
	}

	// 全量抽样往返
	for id := int32(0); id < cfg.MapCount(); id += 97 {
		x, y := cfg.MapID2XY(cores_declarations.MapID(id))
		back := cfg.XY2MapID(x, y)
		if back != cores_declarations.MapID(id) {
			t.Fatalf("round trip failed id=%d → (%d,%d) → %d", id, x, y, back)
		}
	}

	// 边界
	if x, y := cfg.MapID2XY(0); x != 0 || y != 0 {
		t.Fatalf("MapID2XY(0) = (%d,%d), want (0,0)", x, y)
	}
	if x, y := cfg.MapID2XY(cores_declarations.MapID(cfg.MapCount() - 1)); x != 999 || y != 999 {
		t.Fatalf("MapID2XY(%d) = (%d,%d), want (999,999)", cfg.MapCount()-1, x, y)
	}
}

// 2. 格子注册：全量格子均存在且坐标正确
func TestMapDataManager_AllCellsRegistered(t *testing.T) {
	cfg := NewDefaultMapConfig()
	mdm := map_datas.NewMapDataManager(cfg, "map_data")

	for id := int32(0); id < cfg.MapCount(); id += 251 {
		info, ok := mdm.GetMapInfo(cores_declarations.MapID(id))
		if !ok || info == nil {
			t.Fatalf("cell map_id=%d not registered", id)
		}
		x, y := cfg.MapID2XY(cores_declarations.MapID(id))
		if info.GetPointX() != int(x) || info.GetPointY() != int(y) {
			t.Fatalf("cell map_id=%d coord=(%d,%d), want (%d,%d)",
				id, info.GetPointX(), info.GetPointY(), x, y)
		}
	}
}

// 3. 元素生成：无 None 残留，资源格必带 configID
func TestInitMapElements_AllAssigned(t *testing.T) {
	mdm := newTestMapData(t)

	mdm.Range(func(m *map_datas.MapInfo) bool {
		et := m.GetElementType()
		if et == cores_declarations.ElementType_None {
			t.Fatalf("cell map_id=%d has no element assigned", m.GetMapID())
		}
		if et >= cores_declarations.ElementType_Resources_1 && et <= cores_declarations.ElementType_Resources_4 {
			if m.GetElementID() == 0 {
				t.Fatalf("resource cell map_id=%d missing element_id", m.GetMapID())
			}
		}
		return true
	})
}

// 4. 分布：各元素占比贴近权重（确定性种子，宽松容差）
func TestInitMapElements_Distribution(t *testing.T) {
	mdm := newTestMapData(t)

	total := 0
	stats := map[int32]int{}
	mdm.Range(func(m *map_datas.MapInfo) bool {
		stats[int32(m.GetElementType())]++
		total++
		return true
	})

	// 新设计：80% 地形（按权重 45/25/10）+ 20% 资源（四种类型均匀，各 5%）
	expected := map[int32]float64{
		5: 45, // Terrain_1
		6: 25, // Terrain_2
		7: 10, // Terrain_3
		1: 5,  // Resources_1
		2: 5,  // Resources_2
		3: 5,  // Resources_3
		4: 5,  // Resources_4
	}
	for et, pct := range expected {
		got := float64(stats[et]) / float64(total) * 100
		if got < pct-2 || got > pct+2 {
			t.Fatalf("element %d = %.2f%% (%d/%d), want ~%.0f%%", et, got, stats[et], total, pct)
		}
	}
}

// 5. 确定性：同种子两次生成逐格一致
func TestInitMapElements_Deterministic(t *testing.T) {
	a := newTestMapData(t)
	b := newTestMapData(t)

	a.Range(func(m *map_datas.MapInfo) bool {
		info, ok := b.GetMapInfo(m.GetMapID())
		if !ok || info == nil {
			t.Fatalf("cell map_id=%d missing in second map", m.GetMapID())
		}
		if info.GetElementType() != m.GetElementType() || info.GetElementID() != m.GetElementID() {
			t.Fatalf("cell map_id=%d differs: (%d,%d) vs (%d,%d)",
				m.GetMapID(), m.GetElementType(), m.GetElementID(),
				info.GetElementType(), info.GetElementID())
		}
		return true
	})
}

// 6. 链路完整性核心：服务端程序化生成结果 与 模拟 JSON（api/map_data_json）逐格一致
func TestInitMapElements_MatchesSimulatedJSON(t *testing.T) {
	const jsonPath = "../../../../api/map_data_json/map_data_101x101_500_500.json"

	data, err := os.ReadFile(jsonPath)
	if os.IsNotExist(err) {
		t.Skipf("simulated json not found: %s", jsonPath)
	}
	if err != nil {
		t.Fatalf("read json: %v", err)
	}

	var doc struct {
		Cells []struct {
			MapID       int32  `json:"map_id"`
			X           int32  `json:"x"`
			Y           int32  `json:"y"`
			ElementType int32  `json:"element_type"`
			ElementID   uint32 `json:"element_id"`
			Level       int32  `json:"level"`
		} `json:"cells"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if len(doc.Cells) == 0 {
		t.Fatal("json contains no cells")
	}

	mdm := newTestMapData(t)
	for i, c := range doc.Cells {
		info, ok := mdm.GetMapInfo(cores_declarations.MapID(c.MapID))
		if !ok || info == nil {
			t.Fatalf("cell[%d] map_id=%d not found in server map", i, c.MapID)
		}
		if int32(info.GetElementType()) != c.ElementType {
			t.Fatalf("cell[%d] map_id=%d element_type=%d, json=%d", i, c.MapID, info.GetElementType(), c.ElementType)
		}
		if info.GetElementID() != c.ElementID {
			t.Fatalf("cell[%d] map_id=%d element_id=%d, json=%d", i, c.MapID, info.GetElementID(), c.ElementID)
		}
		if int32(info.GetLevel()) != c.Level {
			t.Fatalf("cell[%d] map_id=%d level=%d, json=%d", i, c.MapID, info.GetLevel(), c.Level)
		}
		if info.GetPointX() != int(c.X) || info.GetPointY() != int(c.Y) {
			t.Fatalf("cell[%d] map_id=%d coord=(%d,%d), json=(%d,%d)",
				i, c.MapID, info.GetPointX(), info.GetPointY(), c.X, c.Y)
		}
	}
	t.Logf("matched %d cells against simulated json", len(doc.Cells))
}
