package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// TestHeroAwaken 觉醒：等级不足→专属错误码；无资源→扣减失败；达级+扣资源→成功；重复觉醒→失败
func TestHeroAwaken(t *testing.T) {
	role := game_roles.NewTest(50002)
	hero := role.GetHeroes().AddHero(1)

	// 等级不足（lv1 < awaken_level 20）
	err := HeroAwaken(role, hero)
	if res, ok := err.(rpc_results.ResultI); !ok || res.Code() != pb_error_code.ErrorCode_HeroAwakenLevelNotEnough {
		t.Fatalf("low-level awaken err = %v, want HeroAwakenLevelNotEnough", err)
	}

	// 升到 20 级（1→20 需 NeedExp(1..19) = 100×190 = 19000）
	if _, err := HeroAddExp(hero, 19000); err != nil {
		t.Fatalf("HeroAddExp failed: %v", err)
	}

	// 无资源 → 扣减不足失败
	if err := HeroAwaken(role, hero); err == nil {
		t.Fatal("want error for no resources, got nil")
	}

	// 注入觉醒资源（木/石/粮各 500、铁 300）
	for _, id := range []pb_confs.ItemID{
		pb_confs.ResourceWoodConfID, pb_confs.ResourceStoneConfID, pb_confs.ResourceFoodConfID,
	} {
		role.GetItems().AddItem(common_declarations.ItemUse{
			ItemID: id, ItemType: pb_confs.ItemTypeResource, Count: 500,
		}, time.Now().Unix())
	}
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: pb_confs.ResourceIronConfID, ItemType: pb_confs.ItemTypeResource, Count: 300,
	}, time.Now().Unix())

	if err := HeroAwaken(role, hero); err != nil {
		t.Fatalf("awaken failed: %v", err)
	}
	if !hero.GetIsAwakened() {
		t.Fatal("hero should be awakened")
	}
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceWoodConfID)); got != 0 {
		t.Errorf("wood remain = %d, want 0", got)
	}

	// 重复觉醒 → 已觉醒错误码
	err = HeroAwaken(role, hero)
	if res, ok := err.(rpc_results.ResultI); !ok || res.Code() != pb_error_code.ErrorCode_HeroAlreadyAwakened {
		t.Fatalf("re-awaken err = %v, want HeroAlreadyAwakened", err)
	}
}
