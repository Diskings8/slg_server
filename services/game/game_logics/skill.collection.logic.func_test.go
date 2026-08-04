package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// collectionRole 构造测试角色并造道具
func collectionRole(t *testing.T, items map[pb_confs.ItemID]int64) *game_roles.Role {
	t.Helper()
	role := game_roles.NewTest(50001)
	for id, n := range items {
		role.GetItems().AddItem(common_declarations.ItemUse{ItemID: id, Count: n}, time.Now().Unix())
	}
	return role
}

// TestSkillCollectionActivate_ProgressAndUnlock 分次收集，进度累积，全部达标解锁
func TestSkillCollectionActivate_ProgressAndUnlock(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{2001: 5, 2002: 3}) // 101 需 2001×5 + 2002×3

	// 第一次：消耗 2001×2，进度未达标
	if err := SkillCollectionActivate(role, 101, 2001, 2); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	c := role.GetSkillCollections().GetBySkillConfID(101)
	if c == nil {
		t.Fatal("collection not created")
	}
	if c.IsUnlocked {
		t.Fatal("should not unlock yet (partial progress)")
	}
	if got := collectionCollected(c.CollectionLevel, 2001); got != 2 {
		t.Errorf("collected 2001 = %d, want 2", got)
	}

	// 补足剩余：2001×3 + 2002×3 → 解锁
	if err := SkillCollectionActivate(role, 101, 2001, 3); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if err := SkillCollectionActivate(role, 101, 2002, 3); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	c = role.GetSkillCollections().GetBySkillConfID(101)
	if !c.IsUnlocked {
		t.Fatal("should unlock after all items collected")
	}
	// 道具扣减
	if role.GetItems().GetItemCount(2001) != 0 || role.GetItems().GetItemCount(2002) != 0 {
		t.Error("items should be consumed")
	}
	// 养成消耗记录：3 次激活各记一条（2001×2 / 2001×3 / 2002×3）
	costs := role.GetCultivateCosts().List
	if len(costs) != 3 {
		t.Fatalf("cultivate cost count = %d, want 3", len(costs))
	}
	if costs[0].CultivateType != pb_cultivate.CultivateType_CultivateSkill {
		t.Errorf("cultivate type = %v, want CultivateSkill", costs[0].CultivateType)
	}
	if len(costs[0].Cost) != 1 || costs[0].Cost[0].GetKey() != 2001 || costs[0].Cost[0].GetVal() != 2 {
		t.Errorf("first cultivate cost = %+v, want [{key:2001 val:2}]", costs[0].Cost)
	}
}

// TestSkillCollectionActivate_Unlocked 已解锁不可再激活
func TestSkillCollectionActivate_Unlocked(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{2001: 10, 2002: 3})
	SkillCollectionActivate(role, 101, 2001, 5)
	SkillCollectionActivate(role, 101, 2002, 3)

	if err := SkillCollectionActivate(role, 101, 2001, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionUnlocked) {
		t.Fatalf("err = %v, want ErrSkillCollectionUnlocked", err)
	}
}

// TestSkillCollectionActivate_ItemInvalid 消耗非配置所需道具
func TestSkillCollectionActivate_ItemInvalid(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{2001: 10})
	if err := SkillCollectionActivate(role, 101, 9999, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionItemInvalid) {
		t.Fatalf("err = %v, want ErrSkillCollectionItemInvalid", err)
	}
}

// TestSkillCollectionActivate_ItemFull 该道具已收集满，不可再消耗
func TestSkillCollectionActivate_ItemFull(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{2001: 10, 2002: 5})
	SkillCollectionActivate(role, 101, 2001, 5) // 2001 需求 5，已满

	if err := SkillCollectionActivate(role, 101, 2001, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionItemFull) {
		t.Fatalf("err = %v, want ErrSkillCollectionItemFull", err)
	}
	// 2002 未满仍可消耗（未解锁前）
	if err := SkillCollectionActivate(role, 101, 2002, 1); err != nil {
		t.Fatalf("2002 should be consumable: %v", err)
	}
}

// TestSkillCollectionActivate_ConfNotFound 配置不存在
func TestSkillCollectionActivate_ConfNotFound(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{2001: 10})
	if err := SkillCollectionActivate(role, 9999, 2001, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_SkillCollectionConfNotFound) {
		t.Fatalf("err = %v, want ErrSkillCollectionConfNotFound", err)
	}
}

// TestSkillCollectionActivate_NotEnough 道具不足
func TestSkillCollectionActivate_NotEnough(t *testing.T) {
	role := collectionRole(t, map[pb_confs.ItemID]int64{}) // 无道具
	err := SkillCollectionActivate(role, 101, 2001, 1)
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
