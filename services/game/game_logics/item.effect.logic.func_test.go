package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/models"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_models"
)

// effectRole 构造测试角色：一个英雄 + 若干道具
func effectRole(t *testing.T, items map[pb_confs.ItemID]int64) (*game_roles.Role, *role_heroes.RoleHero) {
	t.Helper()
	role := game_roles.NewTest(40001)
	heroModel := &game_models.RoleHero{ModelBase: models.ModelBase{ID: 1001}, RoleID: role.ID}
	role.GetHeroes().List = append(role.GetHeroes().List, heroModel)
	role.GetHeroes().Init() // 重建 Mem 索引（GetHero 走 Mem）
	for id, n := range items {
		role.GetItems().AddItem(common_declarations.ItemUse{ItemID: id, Count: n}, time.Now().Unix())
	}
	return role, role_heroes.NewRoleHero(heroModel)
}

func itemNum(role *game_roles.Role, id pb_confs.ItemID) int64 {
	return role.GetItems().GetItemCount(int32(id))
}

// TestApplyItemEffect_AddHeroExp 经验书 → 给英雄加经验
func TestApplyItemEffect_AddHeroExp(t *testing.T) {
	role, hero := effectRole(t, map[pb_confs.ItemID]int64{2001: 10}) // 10 张经验书

	if err := ApplyItemEffect(role, 2001, 2, int64(hero.GetID())); err != nil {
		t.Fatalf("use failed: %v", err)
	}
	// 2 张 × 100 exp = 200 exp：level1(need100) → level2, 剩 100
	if hero.GetLevel() != 2 {
		t.Errorf("level = %d, want 2", hero.GetLevel())
	}
	if hero.GetExp() != 100 {
		t.Errorf("exp = %d, want 100", hero.GetExp())
	}
	if itemNum(role, 2001) != 8 {
		t.Errorf("item 2001 remain = %d, want 8", itemNum(role, 2001))
	}
}

// TestApplyItemEffect_AddHeroExp_NoTarget 加经验但目标英雄不存在 → 报错且道具不扣（前置校验）
func TestApplyItemEffect_AddHeroExp_NoTarget(t *testing.T) {
	role, _ := effectRole(t, map[pb_confs.ItemID]int64{2001: 10})

	err := ApplyItemEffect(role, 2001, 1, 9999) // 目标英雄不存在
	if !assertErrorCode(t, err, pb_error_code.ErrorCode_ItemEffectTargetInvalid) {
		t.Fatalf("err = %v, want ErrItemEffectTargetHero", err)
	}
	if itemNum(role, 2001) != 10 {
		t.Errorf("item 2001 remain = %d, want 10 (not consumed)", itemNum(role, 2001))
	}
}

// TestApplyItemEffect_AddCurrency 金币礼包 → 加二级货币
func TestApplyItemEffect_AddCurrency(t *testing.T) {
	role, _ := effectRole(t, map[pb_confs.ItemID]int64{2002: 1})

	if err := ApplyItemEffect(role, 2002, 1, 0); err != nil {
		t.Fatalf("use failed: %v", err)
	}
	if got := itemNum(role, pb_confs.Currency2ConfID); got != 1000 {
		t.Errorf("currency2 = %d, want 1000", got)
	}
	if itemNum(role, 2002) != 0 {
		t.Errorf("item 2002 remain = %d, want 0", itemNum(role, 2002))
	}
}

// TestApplyItemEffect_AddItem 资源包 → 加目标道具
func TestApplyItemEffect_AddItem(t *testing.T) {
	role, _ := effectRole(t, map[pb_confs.ItemID]int64{2003: 1})

	if err := ApplyItemEffect(role, 2003, 1, 0); err != nil {
		t.Fatalf("use failed: %v", err)
	}
	// 2003 → 5 张经验书 2001
	if got := itemNum(role, 2001); got != 5 {
		t.Errorf("item 2001 after = %d, want 5", got)
	}
}

// TestApplyItemEffect_ConfNotFound 配置不存在
func TestApplyItemEffect_ConfNotFound(t *testing.T) {
	role, _ := effectRole(t, map[pb_confs.ItemID]int64{})
	if err := ApplyItemEffect(role, 9999, 1, 0); !assertErrorCode(t, err, pb_error_code.ErrorCode_ItemEffectConfNotFound) {
		t.Fatalf("err = %v, want ErrItemEffectConfNotFound", err)
	}
}

// TestApplyItemEffect_NotEnough 道具不足
func TestApplyItemEffect_NotEnough(t *testing.T) {
	role, _ := effectRole(t, map[pb_confs.ItemID]int64{}) // 无道具
	err := ApplyItemEffect(role, 2001, 1, 1001)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	r, ok := err.(rpc_results.ResultI)
	if !ok {
		t.Fatalf("err type = %T, want rpc_results.ResultI", err)
	}
	if r.Code() != pb_error_code.ErrorCode_ItemTypeNormalNotEnough {
		t.Errorf("code = %d, want ItemTypeNormalNotEnough", r.Code())
	}
}
