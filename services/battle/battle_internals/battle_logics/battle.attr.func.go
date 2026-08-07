package battle_logics

// 英雄战斗属性 — 快照组件求和。
// 属性由 game 侧维护（Cultivate.cur_val = 等级派生基础属性；add_val_* = 各来源加成），
// battle 只做纯算术求和，不再读英雄配置表。将来战时效果（降属性/增益）在此挂修饰符。

import (
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
)

// slotHeroAttr 单个英雄战斗属性（快照派生）
type slotHeroAttr struct {
	Attack       uint32 // 攻击
	Defense      uint32 // 防御
	Intelligence uint32 // 智力
	Movement     uint32 // 移动
	Relocation   uint32 // 拆迁
}

// slotAttr 由队伍快照的英雄 Cultivate 组件求和计算战斗属性。
//
//	effective = cur_val + add_val_camp + add_val_treasure + add_val_features + add_val_city + add_val_title
//
// 各 add_val_* 当前仅 add_val_camp（加点）有值，其余为预留（宝物/特性/城市/称号）。
func slotAttr(slot *pb_battle.TeamSlotInfo) slotHeroAttr {
	hi := slot.GetHeroInfo()
	if hi == nil {
		return slotHeroAttr{}
	}
	return slotHeroAttr{
		Attack:       attrSum(hi.GetAttrAttack()),
		Defense:      attrSum(hi.GetAttrDefense()),
		Intelligence: attrSum(hi.GetAttrIntelligence()),
		Movement:     attrSum(hi.GetAttrMovement()),
		Relocation:   attrSum(hi.GetAttrRelocation()),
	}
}

// attrSum 单维组件求和（nil 按 0）
func attrSum(c *pb_cultivate.Cultivate) uint32 {
	if c == nil {
		return 0
	}
	return c.GetCurVal() + c.GetAddValCamp() + c.GetAddValTreasure() +
		c.GetAddValFeatures() + c.GetAddValCity() + c.GetAddValTitle()
}

// teamAttack 攻击力 = Σ 未受伤英雄 攻击 × 存活士兵
func teamAttack(slots []*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, s := range slots {
		if s == nil || s.GetHeroInfo() == nil {
			continue
		}
		if s.GetHeroInfo().GetCurStatus() == pb_hero.Status_Injured {
			continue
		}
		sum += uint64(slotAttr(s).Attack) * uint64(slotAliveNum(s))
	}
	return sum
}

// teamDefense 防御力 = Σ 未受伤英雄 防御 × 存活士兵
func teamDefense(slots []*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, s := range slots {
		if s == nil || s.GetHeroInfo() == nil {
			continue
		}
		if s.GetHeroInfo().GetCurStatus() == pb_hero.Status_Injured {
			continue
		}
		sum += uint64(slotAttr(s).Defense) * uint64(slotAliveNum(s))
	}
	return sum
}

// teamAttackTeams 多队伍（防守方各行军）总攻击力
func teamAttackTeams(teams [][]*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, team := range teams {
		sum += teamAttack(team)
	}
	return sum
}

// teamDefenseTeams 多队伍（防守方各行军）总防御力
func teamDefenseTeams(teams [][]*pb_battle.TeamSlotInfo) uint64 {
	var sum uint64
	for _, team := range teams {
		sum += teamDefense(team)
	}
	return sum
}

// slotAliveNum 单个槽位当前有效兵力（SoldierInfo/HeroInfo nil 时按 0）
func slotAliveNum(s *pb_battle.TeamSlotInfo) uint32 {
	if s == nil || s.GetHeroInfo() == nil || s.GetHeroInfo().GetSoldierInfo() == nil {
		return 0
	}
	return s.GetHeroInfo().GetSoldierInfo().GetCurAliveNum()
}
