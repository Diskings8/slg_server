package building

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// TestGetBuilding_Embedded 内嵌占位：主城/校场/兵营存在，未配置类型不存在
func TestGetBuilding_Embedded(t *testing.T) {
	c := New()
	for _, typ := range []pb_city.BuildingType{
		pb_city.BuildingType_RoleMainCity,
		pb_city.BuildingType_RoleDrill,
		pb_city.BuildingType_RoleBarracks,
		pb_city.BuildingType_RoleWall,
		pb_city.BuildingType_RoleWarehouse,
		pb_city.BuildingType_RoleFarm,
		pb_city.BuildingType_RoleLumber,
		pb_city.BuildingType_RoleStone,
		pb_city.BuildingType_RoleIron,
	} {
		if _, ok := c.GetBuilding(typ); !ok {
			t.Errorf("building type %d not found", typ)
		}
	}
	if _, ok := c.GetBuilding(pb_city.BuildingType_RoleBranchCity); ok {
		t.Error("branch_city should NOT be in building conf (map overlay)")
	}
}

// TestQueueNumAtLevel 校场队列数断点前向填充
func TestQueueNumAtLevel(t *testing.T) {
	c := New()
	cases := []struct {
		level uint32
		want  uint32
	}{
		{1, 1},
		{2, 2},
		{4, 2}, // 2~4 级均 2 队列
		{5, 3},
		{10, 3}, // 超出最高断点取末值
		{0, 1},  // 无等级默认 1
	}
	for _, tc := range cases {
		if got := c.QueueNumAtLevel(tc.level); got != tc.want {
			t.Errorf("QueueNumAtLevel(%d)=%d, want %d", tc.level, got, tc.want)
		}
	}
}

// TestUpgradeCost 升级消耗曲线 base × growth^(curLevel-1)，ceil
func TestUpgradeCost(t *testing.T) {
	c := New()
	// 主城 1→2：四项资源各 500/500/500/300（木/石/粮/铁），× 1.6^0
	cost, ok := c.UpgradeCost(pb_city.BuildingType_RoleMainCity, 1)
	if !ok || len(cost) != 4 {
		t.Fatalf("UpgradeCost(main,1) failed ok=%v len=%d", ok, len(cost))
	}
	want := []int64{500, 500, 500, 300}
	for i := range want {
		if cost[i].ItemType != pb_confs.ItemTypeResource {
			t.Errorf("cost[%d] type = %d, want ItemTypeResource", i, cost[i].ItemType)
		}
		if cost[i].Count != want[i] {
			t.Errorf("cost[%d] = %d, want %d", i, cost[i].Count, want[i])
		}
	}
	// 主城 2→3：每项 × 1.6^1 = 800/800/800/480
	cost, _ = c.UpgradeCost(pb_city.BuildingType_RoleMainCity, 2)
	if cost[0].Count != 800 || cost[3].Count != 480 {
		t.Errorf("main 2→3 cost = %v, want 800/800/800/480", costCounts(cost))
	}
	// 兵营 1→2：300/300/200/150
	cost, _ = c.UpgradeCost(pb_city.BuildingType_RoleBarracks, 1)
	if cost[0].Count != 300 || cost[2].Count != 200 {
		t.Errorf("barracks 1→2 cost = %v, want 300/300/200/150", costCounts(cost))
	}
	// 资源建筑：不消耗自身产出（farm 升级不含粮）
	farmCost, _ := c.UpgradeCost(pb_city.BuildingType_RoleFarm, 1)
	for _, uc := range farmCost {
		if uc.ItemID == pb_confs.ResourceFoodConfID {
			t.Error("farm upgrade should not cost food (produces food)")
		}
	}
}

func costCounts(cost []common_declarations.ItemUse) []int64 {
	out := make([]int64, len(cost))
	for i, c := range cost {
		out[i] = c.Count
	}
	return out
}

// TestValidate_Invalid 校验失败场景
func TestValidate_Invalid(t *testing.T) {
	c := New()
	// 校场 queue_nums 首项非 1
	drill := c.buildingByType[pb_city.BuildingType_RoleDrill]
	drill.QueueNums = []LevelNum{{Level: 2, Num: 2}}
	if err := c.Validate(); err == nil {
		t.Error("want error for drill queue_nums first level != 1, got nil")
	}
	drill.QueueNums = []LevelNum{{Level: 1, Num: 1}, {Level: 2, Num: 2}, {Level: 5, Num: 3}}
	if err := c.Validate(); err != nil {
		t.Errorf("restored drill should validate, got %v", err)
	}
}

// TestJSONLoad 从 JSON 加载后曲线一致
func TestJSONLoad(t *testing.T) {
	jsonData := []byte(`{"buildings":[
		{"type":101,"name":"main_city","footprint":9,"max_level":10,"build_time_ux":300,
		 "build_cost":[],"upgrade_cost_base":[{"item_id":100002,"item_type":2,"count":500}],
		 "upgrade_cost_growth":1.6,"upgrade_time_growth":1.2},
		{"type":105,"name":"drill","footprint":4,"max_level":10,"build_time_ux":60,
		 "build_cost":[{"item_id":100002,"item_type":2,"count":100}],
		 "upgrade_cost_base":[{"item_id":100002,"item_type":2,"count":200}],
		 "upgrade_cost_growth":1.5,"upgrade_time_growth":1.3,
		 "queue_nums":[{"level":1,"num":1},{"level":2,"num":2},{"level":5,"num":3}]}
	]}`)
	c := New()
	if err := c.Load(jsonData); err != nil {
		t.Fatalf("Load err: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("Validate err: %v", err)
	}
	if c.QueueNumAtLevel(2) != 2 {
		t.Errorf("QueueNumAtLevel(2)=%d, want 2", c.QueueNumAtLevel(2))
	}
	cost, ok := c.UpgradeCost(pb_city.BuildingType_RoleMainCity, 1)
	if !ok || cost[0].Count != 500 {
		t.Errorf("main 1→2 cost after JSON = %+v ok=%v", cost, ok)
	}
}
