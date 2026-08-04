package game_roles

import (
	"os"
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
)

// TestMain 初始化雪花 ID（AddItem 生成主键；空配置建节点 0）
func TestMain(m *testing.M) {
	loggers.Init()
	snowflakes.Init()
	os.Exit(m.Run())
}

// TestRoleCurrency 验证货币（一级/二级）统一走背包系统
func TestRoleCurrency(t *testing.T) {
	role := NewTest(2001)

	// 发放一级货币（充值获得）
	role.AddItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeCurrency1, ItemID: pb_confs.Currency1ConfID, Count: 100},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency1ConfID)); got != 100 {
		t.Fatalf("currency1 = %d, want 100", got)
	}

	// 发放二级货币（游戏内消耗）
	role.AddItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeCurrency2, ItemID: pb_confs.Currency2ConfID, Count: 500},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID)); got != 500 {
		t.Fatalf("currency2 = %d, want 500", got)
	}

	// 扣减二级货币
	logs := role.ReduceItem([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeCurrency2, ItemID: pb_confs.Currency2ConfID, Count: 120},
	}, "test", "test", time.Now().Unix())
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID)); got != 380 {
		t.Fatalf("currency2 after reduce = %d, want 380", got)
	}
	// 产销日志 balance 正确（非 Normal 类型也记录）
	if len(logs) != 1 || logs[0].GetBalance() != 380 {
		t.Fatalf("reduce log = %+v, want single log balance 380", logs)
	}

	// 校验充足
	if code := role.CheckItemEnough([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeCurrency2, ItemID: pb_confs.Currency2ConfID, Count: 300},
	}); code != pb_error_code.ErrorCode_NoneErr {
		t.Fatalf("CheckItemEnough currency2(300) code = %d, want NoneErr", code)
	}
	if code := role.CheckItemEnough([]common_declarations.ItemUse{
		{ItemType: pb_confs.ItemTypeCurrency2, ItemID: pb_confs.Currency2ConfID, Count: 500},
	}); code == pb_error_code.ErrorCode_NoneErr {
		t.Fatal("CheckItemEnough currency2(500) should fail")
	}
}
