package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// TestHeroTroopTransform_ConsumesResources 兵种转化：达级 + 已解锁派生兵种 + 扣资源 → 成功
func TestHeroTroopTransform_ConsumesResources(t *testing.T) {
	role := game_roles.NewTest(50003)
	hero := role.GetHeroes().AddHero(1)

	// 注入兵种扩展道具（unlock_item_conf=1001）
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: 1001, ItemType: pb_confs.ItemTypeNormal, Count: 1,
	}, time.Now().Unix())

	// 解锁派生兵种 101
	if err := HeroTroopUnlock(role, hero, 101); err != nil {
		t.Fatalf("unlock failed: %v", err)
	}

	// 等级不足（lv1 < transform_level 10）→ 失败
	if err := HeroTroopTransform(role, hero, 101); err == nil {
		t.Fatal("want error for low level, got nil")
	}

	// 升到 10 级（1→10 需 100×45=4500 exp）
	if _, err := HeroAddExp(hero, 4500); err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}

	// 无资源 → 扣减不足失败
	if err := HeroTroopTransform(role, hero, 101); err == nil {
		t.Fatal("want error for no resources, got nil")
	}

	// 注入转化资源（木/石各300、粮200、铁150）
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.ResourceWoodConfID, ItemType: pb_confs.ItemTypeResource, Count: 300,
	}, time.Now().Unix())
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.ResourceStoneConfID, ItemType: pb_confs.ItemTypeResource, Count: 300,
	}, time.Now().Unix())
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.ResourceFoodConfID, ItemType: pb_confs.ItemTypeResource, Count: 200,
	}, time.Now().Unix())
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.ResourceIronConfID, ItemType: pb_confs.ItemTypeResource, Count: 150,
	}, time.Now().Unix())

	if err := HeroTroopTransform(role, hero, 101); err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	if hero.GetCurTroopTypeID() != 101 {
		t.Errorf("cur troop = %d, want 101", hero.GetCurTroopTypeID())
	}
	// 资源已扣除
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceWoodConfID)); got != 0 {
		t.Errorf("wood remain = %d, want 0", got)
	}
}
