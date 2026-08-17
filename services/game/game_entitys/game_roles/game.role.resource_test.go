package game_roles

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// TestRoleResource 验证资源（ItemTypeResource）走背包存储 + 仓库上限钳制：
// 未达上限正常入账、跨上限钳制、已在上限 0 入账、扣减/校验与货币一致，且 ChangeLog 记录实际增量。
func TestRoleResource(t *testing.T) {
	// 注入固定上限：粮食 1000（测试后还原，避免污染其他用例）
	old := resourceCap
	defer func() { resourceCap = old }()
	SetResourceCapFunc(func(r *Role, configID pb_confs.ItemID) int64 {
		return 1000
	})

	role := NewTest(2001)

	// 未达上限：正常入账，ChangeLog 记录实际增量
	logs := role.AddItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 400},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 400 {
		t.Fatalf("food = %d, want 400", got)
	}
	if len(logs) != 1 || logs[0].GetDelta() != 400 {
		t.Fatalf("add log = %+v, want single log delta 400", logs)
	}

	// 跨上限：钳制到 1000，实际入账 600，ChangeLog 记录 600
	logs = role.AddItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 5000},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 1000 {
		t.Fatalf("food after clamp = %d, want 1000", got)
	}
	if len(logs) != 1 || logs[0].GetDelta() != 600 {
		t.Fatalf("clamp add log = %+v, want delta 600", logs)
	}

	// 已在上限：0 入账
	logs = role.AddItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 100},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 1000 {
		t.Fatalf("food at cap = %d, want 1000", got)
	}
	if len(logs) != 1 || logs[0].GetDelta() != 0 {
		t.Fatalf("at-cap add log = %+v, want delta 0", logs)
	}

	// 扣减：与货币一致
	logs = role.ReduceItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 300},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.ResourceFoodConfID)); got != 700 {
		t.Fatalf("food after reduce = %d, want 700", got)
	}
	if len(logs) != 1 || logs[0].GetBalance() != 700 {
		t.Fatalf("reduce log = %+v, want balance 700", logs)
	}

	// 校验充足
	if code := role.CheckItemEnough([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 500},
	}); code != pb_error_code.ErrorCode_NoneErr {
		t.Fatalf("CheckItemEnough food(500) code = %d, want NoneErr", code)
	}
	if code := role.CheckItemEnough([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: 800},
	}); code == pb_error_code.ErrorCode_NoneErr {
		t.Fatal("CheckItemEnough food(800) should fail")
	}
}
