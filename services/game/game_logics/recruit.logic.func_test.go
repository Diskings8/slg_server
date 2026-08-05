package game_logics

import (
	"testing"
	"time"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// 池 1001 新手池（配置见 api/game_conf/gacha）：
//
//	FreeDaily/CD=86400；单抽 1券/100金币；GuaranteeTimes=10/GuaranteeGroup=3(史诗)/FirstDrop=2(稀有)
//	WishHeros=[2,3]/WishTimes=20
//	掉落组：普通(70)[hero1 w40,item2001 w30,item2002 w30] / 稀有(25)[hero2 w60,hero3 w40] / 史诗(5)[hero4 w50,hero5 w50,命中保底重置]

// randQueue 顺序随机源（测试注入确定性结果）
type randQueue struct{ vals []int64 }

func (q *randQueue) next(min, _ int64) int64 {
	if len(q.vals) == 0 {
		return min
	}
	v := q.vals[0]
	q.vals = q.vals[1:]
	return v
}

// setRecruitRand 替换随机源并保证测试结束后还原
func setRecruitRand(t *testing.T, vals ...int64) {
	t.Helper()
	old := recruitRand
	q := &randQueue{vals: vals}
	recruitRand = q.next
	t.Cleanup(func() { recruitRand = old })
}

// seedGold 灌金币
func seedGold(role *game_roles.Role, gold int64) {
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID:   pb_confs.Currency2ConfID,
		ItemType: pb_confs.ItemTypeCurrency2,
		Count:    gold,
	}, time.Now().Unix())
}

// markWindowUsed 标记本窗口免费/半价已用（强制走指定消耗档位）
func markWindowUsed(role *game_roles.Role, poolID int32, free, half bool) {
	state := role.GetRecruits().EnsureGetPool(uint32(poolID))
	state.WindowID = currentWindow()
	if free {
		state.FreeTimes = 1
	}
	if half {
		state.HalfTimes = 1
	}
}

// TestRecruit_PoolNotFound 池不存在
func TestRecruit_PoolNotFound(t *testing.T) {
	role := game_roles.NewTest(10001)
	if _, err := Recruit(role, 9999, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitPoolNotFound) {
		t.Fatalf("err = %v, want RecruitPoolNotFound", err)
	}
}

// TestRecruit_InvalidTimes 次数非 1/10
func TestRecruit_InvalidTimes(t *testing.T) {
	role := game_roles.NewTest(10001)
	if _, err := Recruit(role, 1001, 5); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitTimesInvalid) {
		t.Fatalf("err = %v, want RecruitTimesInvalid", err)
	}
}

// TestRecruit_FreeFirstDraw 每日免费首抽：无券无钱也免费，首抽走 FirstDrop（稀有组 hero2）
func TestRecruit_FreeFirstDraw(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0) // 首抽不走组随机，仅组内 item 随机用

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.CostType != int32(RecruitCostFree) {
		t.Errorf("cost_type = %d, want Free(0)", result.CostType)
	}
	if len(result.Rewards) != 1 {
		t.Fatalf("reward count = %d, want 1", len(result.Rewards))
	}
	if !result.Rewards[0].GetIsHero() {
		t.Error("first draw should be hero (rare group)")
	}
	state := role.GetRecruits().EnsureGetPool(1001)
	if state.AllTimes != 1 || state.GuardTimes != 1 {
		t.Errorf("all_times = %d guard_times = %d, want 1/1", state.AllTimes, state.GuardTimes)
	}
	if state.FreeTimes != 1 {
		t.Errorf("free_times = %d, want 1 (free used this window)", state.FreeTimes)
	}
	if len(role.GetHeroes().List) != 1 {
		t.Errorf("hero count = %d, want 1", len(role.GetHeroes().List))
	}
}

// TestRecruit_SecondDrawNotFree 免费用过后再抽不再免费（半价需金币，无钱 → 不足）
func TestRecruit_SecondDrawNotFree(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)

	if _, err := Recruit(role, 1001, 1); err != nil {
		t.Fatalf("first free draw failed: %v", err)
	}
	// 立即再抽：免费已用、半价需金币但无钱
	if _, err := Recruit(role, 1001, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitCostNotEnough) {
		t.Fatalf("err = %v, want RecruitCostNotEnough (not free again)", err)
	}
}

// TestRecruit_HalfPrice 免费用过后半价：金币减半
func TestRecruit_HalfPrice(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)
	seedGold(role, 500)
	markWindowUsed(role, 1001, true, false) // 免费已用，半价可用

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.CostType != int32(RecruitCostHalf) {
		t.Errorf("cost_type = %d, want Half(1)", result.CostType)
	}
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID)); got != 450 {
		t.Errorf("gold count = %d, want 450 (50 half-price spent)", got)
	}
	state := role.GetRecruits().EnsureGetPool(1001)
	if state.HalfTimes != 1 {
		t.Errorf("half_times = %d, want 1", state.HalfTimes)
	}
}

// TestRecruit_TicketPreferred 有券优先扣券，金币不动
func TestRecruit_TicketPreferred(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)
	seedGold(role, 500)
	markWindowUsed(role, 1001, true, true) // 免费/半价已用，走全价
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID:   pb_confs.ItemID(2004),
		ItemType: pb_confs.ItemTypeNormal,
		Count:    5,
	}, time.Now().Unix())

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.CostType != int32(RecruitCostTicket) {
		t.Errorf("cost_type = %d, want Ticket(1)", result.CostType)
	}
	if got := role.GetItems().GetItemCount(2004); got != 4 {
		t.Errorf("ticket count = %d, want 4", got)
	}
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID)); got != 500 {
		t.Errorf("gold count = %d, want 500 (untouched)", got)
	}
}

// TestRecruit_GoldFallback 无券扣金币
func TestRecruit_GoldFallback(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)
	seedGold(role, 500)
	markWindowUsed(role, 1001, true, true) // 免费/半价已用，走全价

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.CostType != int32(RecruitCostGold) {
		t.Errorf("cost_type = %d, want Gold(2)", result.CostType)
	}
	if got := role.GetItems().GetItemCount(int32(pb_confs.Currency2ConfID)); got != 400 {
		t.Errorf("gold count = %d, want 400 (100 spent)", got)
	}
}

// TestRecruit_NotEnoughCost 免费/半价已用且无券无钱不足
func TestRecruit_NotEnoughCost(t *testing.T) {
	role := game_roles.NewTest(10001)
	markWindowUsed(role, 1001, true, true) // 免费/半价已用，但没钱

	if _, err := Recruit(role, 1001, 1); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitCostNotEnough) {
		t.Fatalf("err = %v, want RecruitCostNotEnough", err)
	}
}

// TestRecruit_GuaranteeReset 保底：GuardTimes 到阈值走史诗组，命中后重置为 0
func TestRecruit_GuaranteeReset(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0) // 史诗组内随机：hero4
	seedGold(role, 1000)
	markWindowUsed(role, 1001, true, true) // 走全价，不影响保底判定
	state := role.GetRecruits().EnsureGetPool(1001)
	state.AllTimes = 5   // 已有多次抽取，避开首抽
	state.GuardTimes = 9 // 再抽 1 次即达阈值 10

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if !result.Rewards[0].GetIsGuard() {
		t.Error("reward should be guard hit")
	}
	if got := state.GuardTimes; got != 0 {
		t.Errorf("guard_times = %d, want 0 (reset after guard)", got)
	}
	if got := role.GetHeroes().List[0].HeroConfID; got != 4 {
		t.Errorf("hero conf_id = %d, want 4 (epic group)", got)
	}
}

// TestRecruit_HeroDrop 权重随机命中普通组英雄
func TestRecruit_HeroDrop(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0, 10) // 组=普通(0<70)，item=hero1(10<40)
	seedGold(role, 1000)
	markWindowUsed(role, 1001, true, true)                // 走全价
	role.GetRecruits().EnsureGetPool(1001).AllTimes = 1   // 非首抽，走权重随机

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if !result.Rewards[0].GetIsHero() {
		t.Error("reward should be hero")
	}
	if len(role.GetHeroes().List) != 1 || role.GetHeroes().List[0].HeroConfID != 1 {
		t.Errorf("hero = %+v, want conf_id 1", role.GetHeroes().List)
	}
}

// TestRecruit_ItemDrop 权重随机命中普通组道具（item2002 金币包）
func TestRecruit_ItemDrop(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0, 70) // 组=普通(0<70)，item=item2002(70<=x<100)
	seedGold(role, 1000)
	markWindowUsed(role, 1001, true, true)              // 走全价
	role.GetRecruits().EnsureGetPool(1001).AllTimes = 1 // 非首抽，走权重随机

	result, err := Recruit(role, 1001, 1)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.Rewards[0].GetIsHero() {
		t.Error("reward should be item")
	}
	if got := role.GetItems().GetItemCount(2002); got != 1 {
		t.Errorf("item 2002 count = %d, want 1", got)
	}
}

// TestRecruit_TenDraw 十连：扣十连券、10 条奖励、10 张英雄卡
func TestRecruit_TenDraw(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t) // 全部返回 0：普通组 + hero1
	role.GetItems().AddItem(common_declarations.ItemUse{
		ItemID:   pb_confs.ItemID(2004),
		ItemType: pb_confs.ItemTypeNormal,
		Count:    10,
	}, time.Now().Unix())

	result, err := Recruit(role, 1001, 10)
	if err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if result.CostType != int32(RecruitCostTicket) {
		t.Errorf("cost_type = %d, want Ticket(1)", result.CostType)
	}
	if len(result.Rewards) != 10 {
		t.Errorf("reward count = %d, want 10", len(result.Rewards))
	}
	if got := role.GetItems().GetItemCount(2004); got != 0 {
		t.Errorf("ticket count = %d, want 0 (10 consumed)", got)
	}
	if got := len(role.GetHeroes().List); got != 10 {
		t.Errorf("hero count = %d, want 10", got)
	}
	state := role.GetRecruits().EnsureGetPool(1001)
	if state.AllTimes != 10 {
		t.Errorf("all_times = %d, want 10", state.AllTimes)
	}
	// 第 10 抽 GuardTimes 恰达阈值 → 保底命中史诗组 → 保底计数重置为 0
	if state.GuardTimes != 0 {
		t.Errorf("guard_times = %d, want 0 (pity hit on 10th draw and reset)", state.GuardTimes)
	}
	if !result.Rewards[9].GetIsGuard() {
		t.Error("10th reward should be guard hit")
	}
}

// TestRecruit_WishProgressIncrement 设置心愿后抽卡心愿进度 +1
func TestRecruit_WishProgressIncrement(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)
	seedGold(role, 1000)
	if err := RecruitSetWish(role, 1001, 2); err != nil {
		t.Fatalf("set wish failed: %v", err)
	}

	if _, err := Recruit(role, 1001, 1); err != nil {
		t.Fatalf("recruit failed: %v", err)
	}
	if got := role.GetRecruits().EnsureGetPool(1001).Wish; got != 1 {
		t.Errorf("wish = %d, want 1", got)
	}
}

// TestRecruitSetWish_InvalidHero 心愿英雄不在可选集合
func TestRecruitSetWish_InvalidHero(t *testing.T) {
	role := game_roles.NewTest(10001)
	if err := RecruitSetWish(role, 1001, 999); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitWishHeroInvalid) {
		t.Fatalf("err = %v, want RecruitWishHeroInvalid", err)
	}
}

// TestRecruitSetWish_ChangeResets 变更心愿重置进度
func TestRecruitSetWish_ChangeResets(t *testing.T) {
	role := game_roles.NewTest(10001)
	state := role.GetRecruits().EnsureGetPool(1001)

	if err := RecruitSetWish(role, 1001, 2); err != nil {
		t.Fatalf("set wish failed: %v", err)
	}
	state.Wish = 5
	if err := RecruitSetWish(role, 1001, 3); err != nil {
		t.Fatalf("set wish failed: %v", err)
	}
	if state.ChooseHero != 3 || state.Wish != 0 {
		t.Errorf("choose_hero = %d wish = %d, want 3/0 (wish reset)", state.ChooseHero, state.Wish)
	}
}

// TestRecruitDrawWish_NotReady 心愿进度未满
func TestRecruitDrawWish_NotReady(t *testing.T) {
	role := game_roles.NewTest(10001)
	if err := RecruitSetWish(role, 1001, 2); err != nil {
		t.Fatalf("set wish failed: %v", err)
	}
	role.GetRecruits().EnsureGetPool(1001).Wish = 5 // < 20
	if _, err := RecruitDrawWish(role, 1001); !assertErrorCode(t, err, pb_error_code.ErrorCode_RecruitWishNotReady) {
		t.Fatalf("err = %v, want RecruitWishNotReady", err)
	}
}

// TestRecruitDrawWish_Success 心愿进度满：发心愿英雄卡并扣进度
func TestRecruitDrawWish_Success(t *testing.T) {
	role := game_roles.NewTest(10001)
	if err := RecruitSetWish(role, 1001, 2); err != nil {
		t.Fatalf("set wish failed: %v", err)
	}
	state := role.GetRecruits().EnsureGetPool(1001)
	state.Wish = 20 // 达标

	hero, err := RecruitDrawWish(role, 1001)
	if err != nil {
		t.Fatalf("draw wish failed: %v", err)
	}
	if hero.GetHeroConfID() != 2 {
		t.Errorf("hero conf_id = %d, want 2", hero.GetHeroConfID())
	}
	if state.Wish != 0 {
		t.Errorf("wish = %d, want 0 (20 consumed)", state.Wish)
	}
	if len(role.GetHeroes().List) != 1 {
		t.Errorf("hero count = %d, want 1", len(role.GetHeroes().List))
	}
}

// TestRecruitPoolsInfo 查询池信息：未抽过池状态为零、免费/半价可用状态正确
func TestRecruitPoolsInfo(t *testing.T) {
	role := game_roles.NewTest(10001)
	pools := RecruitPoolsInfo(role)
	if len(pools) != 2 {
		t.Fatalf("pool count = %d, want 2", len(pools))
	}
	if pools[0].GetName() != "新手池" {
		t.Errorf("pool[0] name = %s, want 新手池", pools[0].GetName())
	}
	if pools[0].GetAllTimes() != 0 || pools[0].GetGuardTimes() != 0 {
		t.Errorf("fresh pool state should be zero, got %+v", pools[0])
	}
	// 新手池：免费+半价可用；英雄池：无免费但半价可用
	if !pools[0].GetFreeRemain() || !pools[0].GetHalfPriceRemain() {
		t.Errorf("新手池 fresh free/half remain should be true, got %+v", pools[0])
	}
	if pools[1].GetFreeRemain() || !pools[1].GetHalfPriceRemain() {
		t.Errorf("英雄池 fresh free_remain=false half_remain=true, got %+v", pools[1])
	}
}

// TestRecruitPoolsInfo_WindowUsed 免费用过后池信息不再显示免费可用
func TestRecruitPoolsInfo_WindowUsed(t *testing.T) {
	role := game_roles.NewTest(10001)
	setRecruitRand(t, 0)
	if _, err := Recruit(role, 1001, 1); err != nil {
		t.Fatalf("free draw failed: %v", err)
	}
	pools := RecruitPoolsInfo(role)
	if pools[0].GetFreeRemain() {
		t.Error("free_remain should be false after free draw used this window")
	}
	if !pools[0].GetHalfPriceRemain() {
		t.Error("half_price_remain should still be true (half not used)")
	}
}
