package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// TestSettleRoleResources_Production 惰性结算：资源地按 elapsed×产量入账，结算后重置计时点
func TestSettleRoleResources_Production(t *testing.T) {
	role := game_roles.NewTest(70001)
	roleID := role.ID

	// 1 小时前占领 lv5 资源地（Resources_1=粮食，lv5 产量 1200/h）
	lastUx := time.Now().Unix() - 3600
	role.GetResourceTiles().Upsert(roleID, 10001, 5, int32(cores_declarations.ElementType_Resources_1), lastUx)

	changed := SettleRoleResources(role, roleID)
	if !changed {
		t.Fatal("SettleRoleResources should report change")
	}
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 1200 {
		t.Fatalf("food = %d, want 1200", got)
	}
	// 结算后重置计时点（下一段从此刻起算，不再重复产）
	if changed = SettleRoleResources(role, roleID); changed {
		t.Fatal("immediate second settle should not produce again")
	}
	if tile := role.GetResourceTiles().Get(10001); tile == nil || tile.LastSettleUx <= lastUx {
		t.Fatalf("last settle not advanced: %+v", tile)
	}
}

// TestSettleRoleResources_ZeroElapsed 无时间间隔不产出
func TestSettleRoleResources_ZeroElapsed(t *testing.T) {
	role := game_roles.NewTest(70002)
	role.GetResourceTiles().Upsert(role.ID, 10002, 5, int32(cores_declarations.ElementType_Resources_2), time.Now().Unix())
	if changed := SettleRoleResources(role, role.ID); changed {
		t.Fatal("no elapsed time, should not produce")
	}
}

// TestSyncResourceTile_LevelChange 等级变更：先按旧等级结算再更新，不把整段按新速率误算
func TestSyncResourceTile_LevelChange(t *testing.T) {
	role := game_roles.NewTest(70003)
	roleID := role.ID

	// 1 小时前 lv3（产量 360/h），随后开发 +3 → lv6（1400/h）
	lastUx := time.Now().Unix() - 3600
	role.GetResourceTiles().Upsert(roleID, 10003, 3, int32(cores_declarations.ElementType_Resources_1), lastUx)

	SyncResourceTile(role, roleID, 10003, 6, int32(cores_declarations.ElementType_Resources_1))

	// 旧等级 lv3 结算 360；快照更新为 lv6，计时点重置
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 360 {
		t.Fatalf("food after level-change settle = %d, want 360", got)
	}
	if tile := role.GetResourceTiles().Get(10003); tile == nil || tile.Level != 6 {
		t.Fatalf("tile level not updated: %+v", tile)
	}
}

// TestSyncResourceTile_Remove 非资源元素 → 移除快照（不再产出）
func TestSyncResourceTile_Remove(t *testing.T) {
	role := game_roles.NewTest(70004)
	roleID := role.ID
	role.GetResourceTiles().Upsert(roleID, 10004, 5, int32(cores_declarations.ElementType_Resources_1), time.Now().Unix())
	SyncResourceTile(role, roleID, 10004, 0, int32(cores_declarations.ElementType_Terrain_1))
	if got := role.GetResourceTiles().Get(10004); got != nil {
		t.Fatalf("tile should be removed: %+v", got)
	}
}

// TestSettleRoleResources_MixedLv1 lv1 混合型：全 4 项各产 Amount
func TestSettleRoleResources_MixedLv1(t *testing.T) {
	role := game_roles.NewTest(70005)
	lastUx := time.Now().Unix() - 3600
	role.GetResourceTiles().Upsert(role.ID, 10005, 1, int32(cores_declarations.ElementType_Resources_1), lastUx)

	if !SettleRoleResources(role, role.ID) {
		t.Fatal("should produce")
	}
	for _, id := range []pb_confs.ItemID{
		pb_confs.ResourceFoodConfID, pb_confs.ResourceWoodConfID,
		pb_confs.ResourceStoneConfID, pb_confs.ResourceIronConfID,
	} {
		if got := role.GetItems().GetItemCount(int32(id)); got != 36 {
			t.Fatalf("resource %d = %d, want 36", id, got)
		}
	}
}

// TestSettleRoleResources_DualLv2 lv2 双资源：主资源 + 次级（mapID 稳定派生，恒非主资源）
func TestSettleRoleResources_DualLv2(t *testing.T) {
	role := game_roles.NewTest(70006)
	const mapID int32 = 10006
	lastUx := time.Now().Unix() - 3600
	// 主资源 = 木材 (Resources_2)
	role.GetResourceTiles().Upsert(role.ID, mapID, 2, int32(cores_declarations.ElementType_Resources_2), lastUx)

	if !SettleRoleResources(role, role.ID) {
		t.Fatal("should produce")
	}
	sec := dualSecondaryResourceID(pb_confs.ResourceWoodConfID, mapID)
	if sec == pb_confs.ResourceWoodConfID {
		t.Fatalf("secondary should differ from main")
	}
	for _, id := range []pb_confs.ItemID{
		pb_confs.ResourceFoodConfID, pb_confs.ResourceWoodConfID,
		pb_confs.ResourceStoneConfID, pb_confs.ResourceIronConfID,
	} {
		want := int64(0)
		if id == pb_confs.ResourceWoodConfID || id == sec {
			want = 120
		}
		if got := role.GetItems().GetItemCount(int32(id)); got != want {
			t.Fatalf("resource %d = %d, want %d", id, got, want)
		}
	}
}
