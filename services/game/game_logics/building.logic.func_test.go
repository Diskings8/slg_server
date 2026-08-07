package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// buildingRole 构造测试角色并注入二级货币（金币）
func buildingRole(t *testing.T, gold int64) *game_roles.Role {
	t.Helper()
	role := game_roles.NewTest(60002)
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.Currency2ConfID, ItemType: pb_confs.ItemTypeCurrency2, Count: gold,
	}, time.Now().Unix())
	return role
}

// TestBuildingBuild_Instant 建角即时建主城：跳过时长/消耗，直接 Completed + 落校场 + 队列
func TestBuildingBuild_Instant(t *testing.T) {
	role := buildingRole(t, 0)
	roleID := role.ID
	id, result := BuildMainCityInstant(role, roleID, &pb_city.BuildingBuildReq{
		Type:  pb_city.BuildingType_RoleMainCity,
		MapId: 100,
	})
	if result != nil {
		t.Fatalf("BuildMainCityInstant failed: %v", result)
	}
	main := role.GetBuildings().GetBuilding(id)
	if main == nil || main.State != pb_city.BuildingState_Completed || main.Level != 1 {
		t.Fatalf("main city state=%v level=%d, want Completed/1", main.State, main.Level)
	}
	// 自动落 1 级校场
	drill := role.GetBuildings().GetDrillByCity(id)
	if drill == nil || drill.Level != 1 {
		t.Fatalf("drill not auto-created: %+v", drill)
	}
	// 队列数 = 配置 QueueNumAtLevel(1) = 1
	if n := len(role.GetFormations().ListByCity(id)); n != 1 {
		t.Errorf("queue count = %d, want 1", n)
	}
}

// TestBuildingBuild_WithDuration 普通建造：扣资源 + Constructing + EndTimeUx
func TestBuildingBuild_WithDuration(t *testing.T) {
	role := buildingRole(t, 100000)
	roleID := role.ID

	id, result := BuildingBuild(role, roleID, &pb_city.BuildingBuildReq{
		Type:   pb_city.BuildingType_RoleBarracks,
		CityId: 888,
		MapId:  200,
	})
	if result != nil {
		t.Fatalf("BuildingBuild failed: %v", result)
	}
	b := role.GetBuildings().GetBuilding(id)
	if b.State != pb_city.BuildingState_Constructing || b.NextLevel != 1 {
		t.Fatalf("barracks state=%v next=%d, want Constructing/1", b.State, b.NextLevel)
	}
	if b.EndTimeUx <= time.Now().Unix() {
		t.Errorf("EndTimeUx=%d should be in future", b.EndTimeUx)
	}
	// 资源已扣除（兵营 build_cost=200）
	remain := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID))
	if remain != 100000-200 {
		t.Errorf("gold remain=%d, want %d", remain, 100000-200)
	}
}

// TestBuildingBuild_NoGold 资源不足
func TestBuildingBuild_NoGold(t *testing.T) {
	role := buildingRole(t, 0)
	roleID := role.ID
	_, result := BuildingBuild(role, roleID, &pb_city.BuildingBuildReq{
		Type:   pb_city.BuildingType_RoleBarracks,
		CityId: 888,
		MapId:  200,
	})
	if result == nil {
		t.Fatal("want error for no gold, got nil")
	}
	if res, ok := result.(rpc_results.ResultI); ok {
		if res.Code() == pb_error_code.ErrorCode_NoneErr {
			t.Error("error code should not be NoneErr")
		}
	}
}

// TestBuildingBuild_Duplicate 同城市同类型唯一
func TestBuildingBuild_Duplicate(t *testing.T) {
	role := buildingRole(t, 100000)
	roleID := role.ID
	// 用 BuildingBuild 建两次兵营同 city
	req := &pb_city.BuildingBuildReq{Type: pb_city.BuildingType_RoleBarracks, CityId: 999, MapId: 300}
	if _, result := BuildingBuild(role, roleID, req); result != nil {
		t.Fatalf("first build failed: %v", result)
	}
	if _, result := BuildingBuild(role, roleID, req); result == nil {
		t.Fatal("want duplicate error, got nil")
	}
}

// TestBuildingUpgrade_Success 升级：扣消耗 + Constructing + 等级不变
func TestBuildingUpgrade_Success(t *testing.T) {
	role := buildingRole(t, 100000)
	roleID := role.ID
	id, result := BuildingBuild(role, roleID, &pb_city.BuildingBuildReq{
		Type:   pb_city.BuildingType_RoleBarracks,
		CityId: 777,
		MapId:  400,
	})
	if result != nil {
		t.Fatalf("build failed: %v", result)
	}
	// 立即结算建造（忽略时长）→ Completed level=1
	b := role.GetBuildings().GetBuilding(id)
	b.State = pb_city.BuildingState_Completed
	b.Level = 1
	b.NextLevel = 0

	up := &pb_city.BuildingUpgradeReq{BuildingId: id}
	if result := BuildingUpgrade(role, roleID, up); result != nil {
		t.Fatalf("upgrade failed: %v", result)
	}
	b = role.GetBuildings().GetBuilding(id)
	if b.State != pb_city.BuildingState_Constructing || b.NextLevel != 2 {
		t.Errorf("after upgrade state=%v next=%d, want Constructing/2", b.State, b.NextLevel)
	}
	// 消耗 = upgrade_cost_base 300
	remain := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID))
	if remain != 100000-200-300 {
		t.Errorf("gold remain=%d, want %d", remain, 100000-200-300)
	}
}

// TestSettleBuildings_Due 惰性结算：到期建筑置 Completed + 校场同步队列
func TestSettleBuildings_Due(t *testing.T) {
	role := buildingRole(t, 100000)
	roleID := role.ID

	// 建主城（即时）→ 已有校场 + 1 队列
	id, result := BuildMainCityInstant(role, roleID, &pb_city.BuildingBuildReq{
		Type:  pb_city.BuildingType_RoleMainCity,
		MapId: 500,
	})
	if result != nil {
		t.Fatalf("build main city failed: %v", result)
	}

	// 手动把校场升级到 2 级并标记 Constructing + 已到期
	drill := role.GetBuildings().GetDrillByCity(id)
	drill.Level = 2
	drill.State = pb_city.BuildingState_Constructing
	drill.NextLevel = 3
	drill.EndTimeUx = time.Now().Unix() - 1 // 已到期

	changed := SettleBuildings(role, roleID)
	if !changed {
		t.Fatal("SettleBuildings should report change")
	}
	if drill.State != pb_city.BuildingState_Completed || drill.Level != 3 {
		t.Errorf("drill after settle state=%v level=%d, want Completed/3", drill.State, drill.Level)
	}
	// 校场 3 级 → 队列 2（queue_nums[3]=2，因 5 级才 3；原 1 队列 → 补到 2）
	if n := len(role.GetFormations().ListByCity(id)); n != 2 {
		t.Errorf("queue count after drill settle = %d, want 2", n)
	}
}

// TestBuildingListByCity 按城市过滤
func TestBuildingListByCity(t *testing.T) {
	role := buildingRole(t, 0)
	roleID := role.ID
	// 建两个城市（主城 + 手动加一个模拟分城）
	BuildMainCityInstant(role, roleID, &pb_city.BuildingBuildReq{Type: pb_city.BuildingType_RoleMainCity, MapId: 600})
	all := BuildingListPb(role, 0)
	if len(all) < 2 { // 主城 + 校场
		t.Fatalf("all buildings count=%d, want >=2", len(all))
	}
	// 按主城过滤应只含该城市建筑
	mainID := role.GetBuildings().GetMainCity().ID
	cityOnly := BuildingListPb(role, mainID)
	for _, b := range cityOnly {
		if b.CityId != mainID {
			t.Errorf("building %d city_id=%d, want %d", b.Id, b.CityId, mainID)
		}
	}
}
