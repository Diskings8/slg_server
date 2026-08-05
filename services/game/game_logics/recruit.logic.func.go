package game_logics

import (
	"time"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_recruit"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/common/utils/util_randoms"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_models"
)

// recruitRand 可替换的随机源（单元测试注入确定性结果用）
var recruitRand = util_randoms.BetweenInt64

// RecruitCostType 抽卡消耗类型
type RecruitCostType int32

const (
	RecruitCostFree  RecruitCostType = iota // 免费（本窗口 1 次）
	RecruitCostHalf                        // 半价（本窗口 1 次，金币减半）
	RecruitCostTicket                      // 消耗抽卡券
	RecruitCostGold                        // 消耗金币（二级货币）
)

// RecruitResult 抽卡结果
type RecruitResult struct {
	CostType int32 // 0=免费 1=券 2=金币
	Rewards  []*pb_recruit.RecruitReward
}

// RecruitPoolsInfo 查询所有抽卡池信息（含名称与当前窗口免费/半价可用状态）
//
// 纯读：不会创建池状态，未抽过的池状态字段为零值。
func RecruitPoolsInfo(role *game_roles.Role) []*pb_recruit.RecruitPoolInfo {
	gc := game_conf.Load().Gacha
	list := make([]*pb_recruit.RecruitPoolInfo, 0)
	for _, poolID := range gc.AllPoolIDs() {
		poolConf, ok := gc.GetPool(poolID)
		if !ok {
			continue
		}
		state := role.GetRecruits().GetPool(uint32(poolID))
		info := &pb_recruit.RecruitPoolInfo{
			Id:   poolID,
			Name: poolConf.Name,
		}
		freeUsed, halfUsed := windowUsage(poolConf, state)
		info.FreeRemain = poolConf.FreeDaily && !freeUsed
		info.HalfPriceRemain = poolConf.HalfPrice && !halfUsed
		if state != nil {
			info.AllTimes = state.AllTimes
			info.GuardTimes = state.GuardTimes
			info.Wish = state.Wish
			info.ChooseHero = state.ChooseHero
		}
		list = append(list, info)
	}
	return list
}

// Recruit 抽卡（单抽/十连）
//
// 消耗顺序：免费（本窗口）→ 半价（本窗口）→ 抽卡券 → 金币。产出按权重随机，含保底与心愿进度累计。
func Recruit(role *game_roles.Role, poolID, times int32) (*RecruitResult, error) {
	poolConf, ok := game_conf.Load().Gacha.GetPool(poolID)
	if !ok {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitPoolNotFound, "recruit pool not found")
	}
	if times != 1 && times != 10 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitTimesInvalid, "recruit times must be 1 or 10")
	}

	state := role.GetRecruits().EnsureGetPool(uint32(poolID))
	resetWindow(state)

	// 消耗判断
	costType, err := consumeRecruitCost(role, poolConf, state, times)
	if err != nil {
		return nil, err
	}

	// 逐次抽卡
	rewards := make([]*pb_recruit.RecruitReward, 0, times)
	addItems := make([]common_declarations.ItemUse, 0)
	for i := int32(0); i < times; i++ {
		reward, item, err := executeRecruitOnce(role, poolConf, state, int32(i+1))
		if err != nil {
			return nil, err
		}
		rewards = append(rewards, reward)
		if item != nil {
			addItems = append(addItems, *item)
		}
	}

	// 发放道具产出（英雄卡已由 AddHero 写入实体）
	if len(addItems) > 0 {
		if err := ItemChange(role, addItems, nil, common_declarations.ReasonGacha); err != nil {
			return nil, err
		}
	}

	return &RecruitResult{CostType: int32(costType), Rewards: rewards}, nil
}

// RecruitSetWish 设置心愿英雄（0=取消；变更时重置心愿进度）
func RecruitSetWish(role *game_roles.Role, poolID, chooseHero int32) error {
	poolConf, ok := game_conf.Load().Gacha.GetPool(poolID)
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_RecruitPoolNotFound, "recruit pool not found")
	}
	if chooseHero != 0 && !contains(poolConf.WishHeros, chooseHero) {
		return rpc_results.Error(pb_error_code.ErrorCode_RecruitWishHeroInvalid, "wish hero not in pool wish list")
	}

	state := role.GetRecruits().EnsureGetPool(uint32(poolID))
	if state.ChooseHero != chooseHero {
		state.ChooseHero = chooseHero
		state.Wish = 0 // 变更心愿重置进度
	}
	return nil
}

// RecruitDrawWish 领取心愿英雄卡（进度达标扣进度并发卡）
func RecruitDrawWish(role *game_roles.Role, poolID int32) (*role_heroes.RoleHero, error) {
	poolConf, ok := game_conf.Load().Gacha.GetPool(poolID)
	if !ok {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitPoolNotFound, "recruit pool not found")
	}
	state := role.GetRecruits().EnsureGetPool(uint32(poolID))
	if state.ChooseHero == 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitWishHeroInvalid, "wish hero not set")
	}
	if state.Wish < uint32(poolConf.WishTimes) {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitWishNotReady, "wish progress not ready")
	}

	hero := role.GetHeroes().AddHero(state.ChooseHero)
	state.Wish -= uint32(poolConf.WishTimes)
	return hero, nil
}

// -------------------------------- 内部实现 --------------------------------

// currentWindow 当前免费/半价窗口ID（每天 0/12 点切分两个窗口）
//
// 窗口ID = 本地日序号 * 2 + 时段（0=0~12点，1=12~24点）。
func currentWindow() int64 {
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayNum := dayStart.Unix() / 86400
	half := int64(0)
	if now.Hour() >= 12 {
		half = 1
	}
	return dayNum*2 + half
}

// resetWindow 跨窗口重置免费/半价次数
func resetWindow(state *game_models.RecruitPool) {
	win := currentWindow()
	if state.WindowID != win {
		state.WindowID = win
		state.FreeTimes = 0
		state.HalfTimes = 0
	}
}

// windowUsage 本窗口免费/半价是否已用（跨窗口视为未用）
func windowUsage(poolConf *gacha.RecruitPoolConfig, state *game_models.RecruitPool) (freeUsed, halfUsed bool) {
	if state == nil || state.WindowID != currentWindow() {
		return false, false
	}
	return state.FreeTimes > 0, state.HalfTimes > 0
}

// consumeRecruitCost 判断并扣除本次抽卡消耗
//
// 消耗顺序（仅单抽）：免费 → 半价（金币减半）→ 全价（券优先于金币）。
func consumeRecruitCost(role *game_roles.Role, poolConf *gacha.RecruitPoolConfig, state *game_models.RecruitPool, times int32) (RecruitCostType, error) {
	// 免费：本窗口未用
	if times == 1 && poolConf.FreeDaily && state.FreeTimes == 0 {
		state.FreeTimes++
		return RecruitCostFree, nil
	}

	// 半价：本窗口未用且金币足够
	halfGold := poolConf.SingleGold / 2
	if times == 1 && poolConf.HalfPrice && state.HalfTimes == 0 && halfGold > 0 &&
		role.GetItems().HasItem(int32(pb_confs.Currency2ConfID), halfGold) {
		state.HalfTimes++
		if err := ItemChange(role, nil, []common_declarations.ItemUse{{
			ItemID:   pb_confs.Currency2ConfID,
			ItemType: pb_confs.ItemTypeCurrency2,
			Count:    halfGold,
		}}, common_declarations.ReasonGacha); err != nil {
			return 0, err
		}
		return RecruitCostHalf, nil
	}

	needTicket, needGold := recruitCost(poolConf, times)
	var useItems []common_declarations.ItemUse

	// 抽卡券优先
	if poolConf.TicketConfID != 0 && needTicket > 0 && role.GetItems().HasItem(poolConf.TicketConfID, needTicket) {
		useItems = []common_declarations.ItemUse{{
			ItemID:   pb_confs.ItemID(poolConf.TicketConfID),
			ItemType: pb_confs.ItemTypeNormal,
			Count:    needTicket,
		}}
		if err := ItemChange(role, nil, useItems, common_declarations.ReasonGacha); err != nil {
			return 0, err
		}
		return RecruitCostTicket, nil
	}

	// 金币兜底
	if needGold > 0 && role.GetItems().HasItem(int32(pb_confs.Currency2ConfID), needGold) {
		useItems = []common_declarations.ItemUse{{
			ItemID:   pb_confs.Currency2ConfID,
			ItemType: pb_confs.ItemTypeCurrency2,
			Count:    needGold,
		}}
		if err := ItemChange(role, nil, useItems, common_declarations.ReasonGacha); err != nil {
			return 0, err
		}
		return RecruitCostGold, nil
	}

	return 0, rpc_results.Error(pb_error_code.ErrorCode_RecruitCostNotEnough, "ticket or gold not enough")
}

// recruitCost 单抽/十连的券数与金币数
func recruitCost(poolConf *gacha.RecruitPoolConfig, times int32) (ticket, gold int64) {
	if times == 10 {
		return poolConf.TenTicket, poolConf.TenGold
	}
	return poolConf.SingleTicket, poolConf.SingleGold
}

// executeRecruitOnce 单次抽卡：保底/首抽/权重随机 → 产出英雄卡或道具
//
// 返回奖励信息与待发放道具（英雄卡已直接写入角色英雄集合）。
func executeRecruitOnce(role *game_roles.Role, poolConf *gacha.RecruitPoolConfig, state *game_models.RecruitPool, nth int32) (*pb_recruit.RecruitReward, *common_declarations.ItemUse, error) {
	state.GuardTimes++
	state.AllTimes++
	if state.ChooseHero != 0 {
		state.Wish++
	}

	// 选择掉落组：首抽 → 保底 → 权重随机
	var dropGroup *gacha.DropGroupConfig
	if state.AllTimes == 1 && poolConf.FirstDropGroupID != 0 {
		dropGroup = findGroup(poolConf, poolConf.FirstDropGroupID)
	} else if state.GuardTimes >= uint32(poolConf.GuaranteeTimes) {
		dropGroup = findGroup(poolConf, poolConf.GuaranteeGroupID)
	} else {
		g, err := randomDropGroup(poolConf)
		if err != nil {
			return nil, nil, err
		}
		dropGroup = g
	}
	if dropGroup == nil {
		return nil, nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitDropConfNotFound, "drop group config not found")
	}

	// 组内权重随机
	dropItem, err := randomDropItem(dropGroup)
	if err != nil {
		return nil, nil, err
	}
	if dropItem.GuaranteeReset {
		state.GuardTimes = 0
	}

	reward := &pb_recruit.RecruitReward{
		Nth:     nth,
		DropId:  dropGroup.GroupID,
		IsGuard: dropItem.GuaranteeReset,
	}
	var itemUse *common_declarations.ItemUse
	if dropItem.IsHero {
		heroCard := role.GetHeroes().AddHero(dropItem.RewardConfID)
		reward.IsHero = true
		reward.ConfId = dropItem.RewardConfID
		reward.Count = 1
		reward.HeroId = int32(heroCard.GetID())
	} else {
		itemUse = &common_declarations.ItemUse{
			ItemID:   pb_confs.ItemID(dropItem.RewardConfID),
			ItemType: pb_confs.ItemTypeNormal,
			Count:    int64(dropItem.Count),
		}
		reward.IsHero = false
		reward.ConfId = dropItem.RewardConfID
		reward.Count = dropItem.Count
	}
	return reward, itemUse, nil
}

// findGroup 按组ID查找掉落组
func findGroup(poolConf *gacha.RecruitPoolConfig, groupID int32) *gacha.DropGroupConfig {
	for i := range poolConf.DropGroups {
		if poolConf.DropGroups[i].GroupID == groupID {
			return &poolConf.DropGroups[i]
		}
	}
	return nil
}

// randomDropGroup 按组权重随机一组
func randomDropGroup(poolConf *gacha.RecruitPoolConfig) (*gacha.DropGroupConfig, error) {
	var totalWeight int32
	for _, g := range poolConf.DropGroups {
		totalWeight += g.Weight
	}
	if totalWeight <= 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitDropConfNotFound, "drop group total weight is zero")
	}

	randValue := recruitRand(0, int64(totalWeight))
	var currentWeight int32
	for i := range poolConf.DropGroups {
		g := &poolConf.DropGroups[i]
		currentWeight += g.Weight
		if randValue < int64(currentWeight) {
			return g, nil
		}
	}
	return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitDropConfNotFound, "random drop group failed")
}

// randomDropItem 组内按条目权重随机一条
func randomDropItem(group *gacha.DropGroupConfig) (*gacha.DropItemConfig, error) {
	var totalWeight int32
	for i := range group.Items {
		totalWeight += group.Items[i].Weight
	}
	if totalWeight <= 0 {
		return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitDropConfNotFound, "drop item total weight is zero")
	}

	randValue := recruitRand(0, int64(totalWeight))
	var currentWeight int32
	for i := range group.Items {
		item := &group.Items[i]
		currentWeight += item.Weight
		if randValue < int64(currentWeight) {
			return item, nil
		}
	}
	return nil, rpc_results.Error(pb_error_code.ErrorCode_RecruitDropConfNotFound, "random drop item failed")
}

// contains 判断切片是否包含目标值
func contains(list []int32, target int32) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
