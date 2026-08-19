package gacha

import (
	"testing"

	"server.slg.com/api/protocol/pb/pb_gameconfig"
)

// poolFixture 新手池 pb.Table：掉落组 1（普通）+ 3（保底），含英雄与道具掉落。
func poolFixture() *pb_gameconfig.Table {
	return &pb_gameconfig.Table{
		GachaPool: []*pb_gameconfig.GachaPool{
			{
				PoolId: 1001, Name: "新手池", TicketConfId: 2004,
				SingleTicket: 1, SingleGold: 100, TenTicket: 10, TenGold: 900,
				FreeDaily: true, HalfPrice: true,
				GuaranteeTimes: 10, GuaranteeGroupId: 3, FirstDropGroupId: 0,
				WishHeros: []int32{2, 3}, WishTimes: 20,
			},
		},
		GachaDropGroup: []*pb_gameconfig.GachaDropGroup{
			{PoolId: 1001, GroupId: 1, Weight: 70},
			{PoolId: 1001, GroupId: 3, Weight: 5},
		},
		GachaDropItem: []*pb_gameconfig.GachaDropItem{
			{PoolId: 1001, GroupId: 1, RewardConfId: 1, IsHero: true, Count: 1, Weight: 40},
			{PoolId: 1001, GroupId: 1, RewardConfId: 2001, IsHero: false, Count: 5, Weight: 30},
			{PoolId: 1001, GroupId: 3, RewardConfId: 4, IsHero: true, Count: 1, Weight: 50, GuaranteeReset: true},
		},
	}
}

// TestGacha_LoadAndQuery 经 pb.Table → NewFromPB 构建后抽卡池/掉落组/保底查询正常。
func TestGacha_LoadAndQuery(t *testing.T) {
	c, err := NewFromPB(poolFixture())
	if err != nil {
		t.Fatalf("NewFromPB failed: %v", err)
	}
	pool, ok := c.GetPool(1001)
	if !ok || pool.Name != "新手池" || pool.SingleGold != 100 || pool.GuaranteeTimes != 10 {
		t.Errorf("GetPool(1001) = %+v, ok=%v", pool, ok)
	}
	if len(pool.DropGroups) != 2 || pool.DropGroups[0].Items[0].RewardConfID != 1 {
		t.Errorf("drop groups mismatch: %+v", pool.DropGroups)
	}
	if pool.DropGroups[1].Items[0].GuaranteeReset != true {
		t.Error("guarantee_reset should be true on epic item")
	}
	ids := c.AllPoolIDs()
	if len(ids) != 1 || ids[0] != 1001 {
		t.Errorf("AllPoolIDs = %v, want [1001]", ids)
	}
}

// TestGacha_LoadDuplicateKey pool_id 重复 → NewFromPB 报错。
func TestGacha_LoadDuplicateKey(t *testing.T) {
	tb := &pb_gameconfig.Table{
		GachaPool: []*pb_gameconfig.GachaPool{
			{PoolId: 1001, Name: "a", TicketConfId: 2004, SingleTicket: 1, SingleGold: 100, TenTicket: 10, TenGold: 900,
				FreeDaily: true, HalfPrice: true, GuaranteeTimes: 10, GuaranteeGroupId: 3, FirstDropGroupId: 0, WishTimes: 20},
			{PoolId: 1001, Name: "b", TicketConfId: 2004, SingleTicket: 1, SingleGold: 100, TenTicket: 10, TenGold: 900,
				FreeDaily: true, HalfPrice: true, GuaranteeTimes: 10, GuaranteeGroupId: 3, FirstDropGroupId: 0, WishTimes: 20},
		},
		GachaDropGroup: []*pb_gameconfig.GachaDropGroup{
			{PoolId: 1001, GroupId: 1, Weight: 70},
		},
		GachaDropItem: []*pb_gameconfig.GachaDropItem{
			{PoolId: 1001, GroupId: 1, RewardConfId: 1, IsHero: true, Count: 1, Weight: 40},
		},
	}
	if _, err := NewFromPB(tb); err == nil {
		t.Fatal("NewFromPB should fail on duplicate pool_id")
	}
}

// TestGacha_ValidateGuaranteeGroupMissing 保底组引用不存在的组 → NewFromPB 校验报错。
func TestGacha_ValidateGuaranteeGroupMissing(t *testing.T) {
	tb := &pb_gameconfig.Table{
		GachaPool: []*pb_gameconfig.GachaPool{
			{PoolId: 1001, Name: "a", TicketConfId: 2004, SingleTicket: 1, SingleGold: 100, TenTicket: 10, TenGold: 900,
				FreeDaily: true, HalfPrice: true, GuaranteeTimes: 10, GuaranteeGroupId: 99, FirstDropGroupId: 0, WishTimes: 20},
		},
		GachaDropGroup: []*pb_gameconfig.GachaDropGroup{
			{PoolId: 1001, GroupId: 1, Weight: 70},
		},
		GachaDropItem: []*pb_gameconfig.GachaDropItem{
			{PoolId: 1001, GroupId: 1, RewardConfId: 1, IsHero: true, Count: 1, Weight: 40},
		},
	}
	if _, err := NewFromPB(tb); err == nil {
		t.Fatal("NewFromPB should fail when guarantee_group_id not in drop_groups")
	}
}
