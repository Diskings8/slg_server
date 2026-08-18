package battle_logics

// 8 回合回合制战斗引擎。
//
// 框架（用户设计）：
//   - 战前准备：属性分层求和（Cultivate 组件 → 有效攻/防/智/移/拆 + 攻击距离）+ 技能加载
//   - 每方最多 3 英雄（大营 slot1 / 1号位 / 2号位），站位 [1,2,3,3,2,1]（槽位 1 基）
//   - 8 回合，每回合 开始→行动→结算；存活英雄按移动(速度)降序行动，每人每回合一次
//   - 行动序列：主动技能 → 普攻 → 追击技能（追击按技能配置概率触发）
//   - 普攻/技能目标：攻击范围内（"到对方大营距离 ≤ D"）随机单个
//   - 伤害：物理 攻击²/(攻击+防御×收敛系数)×当前有效兵力；法术用双方智力同式
//   - 伤兵：伤害按受伤比例(第1回合85%，每回合-10%)拆死亡+伤兵；伤兵不计有效兵力；
//     回合结算阶段每回合 10% 当前伤兵转死亡；技能可恢复伤兵
//   - 胜负：大营(slot1)有效兵力==0 → 该方败；同回合同归于尽；8 回合平局

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_skill"
)

// stanceIndex 某侧某槽位的站位索引（站位 [1,2,3,3,2,1]：攻大营/攻1号/攻2号/守2号/守1号/守大营）
// 攻方槽位 1,2,3 → 索引 0,1,2；守方槽位 1,2,3 → 索引 5,4,3
func stanceIndex(side, slot int32) int32 {
	if side == 0 {
		return slot - 1
	}
	return 6 - slot
}

// stanceDistance 攻守两个单位间的站位距离
func stanceDistance(aSide int, aSlot int32, bSide int, bSlot int32) int32 {
	d := stanceIndex(int32(aSide), aSlot) - stanceIndex(int32(bSide), bSlot)
	if d < 0 {
		return -d
	}
	return d
}

// battleUnit 战前准备后的单个战斗单位（有效属性 + 兵力状态）
type battleUnit struct {
	slot        int32  // 槽位（1=大营，2/3=1/2号位）
	side        int    // 0=攻 1=守
	heroID      uint64 // 英雄实例ID
	level       uint32 // 当前等级（经验/战报用）
	attrs       slotHeroAttr
	attackRange uint32 // 攻击距离
	alive       uint32 // 有效兵力（不含伤兵）
	injured     uint32 // 伤兵
	dead        uint32 // 累计死亡兵
	skills      []*pb_skill.Skill
}

// battleCtx 一场战斗上下文
type battleCtx struct {
	rules    *battle.Conf
	attacker []*battleUnit
	defender []*battleUnit
	round    uint32
	step     uint32 // 伪随机步进（目标选择循环偏移）
	finish   bool   // 已分出胜负（含同归/平局提前终止）
	draw     bool   // 8 回合平局
}

// runBattle 攻守双方一场 8 回合战斗，返回 OneBattleResult（含回合日志）。
//
// 入参 attackerSlots/defenderSlots 会被原地更新为战后快照（伤亡写回）。
func runBattle(attackerSlots, defenderSlots []*pb_battle.TeamSlotInfo) *pb_battle.OneBattleResult {
	b := &battleCtx{
		rules:    game_conf.Load().Battle,
		attacker: prepSide(attackerSlots, 0),
		defender: prepSide(defenderSlots, 1),
	}
	if b.rules == nil || b.rules.Rounds == 0 {
		b.rules = battle.New()
	}
	result := &pb_battle.OneBattleResult{
		Attacker: &pb_battle.BattleSide{TeamInfo: &pb_battle.TeamInfo{SlotInfo: attackerSlots}},
		Defender: &pb_battle.BattleSide{TeamInfo: &pb_battle.TeamInfo{SlotInfo: defenderSlots}},
	}

	for r := uint32(1); r <= b.rules.Rounds && !b.finish; r++ {
		b.round = r
		br := &pb_battle.BattleRound{Round: int32(r)}

		// 回合行动阶段：存活英雄按移动降序行动
		for _, u := range b.actionOrder() {
			if u.alive == 0 {
				continue
			}
			br.Actions = append(br.Actions, b.resolveUnitAction(u)...)
		}

		// 回合结算阶段：当前伤兵 10% 转死亡
		br.Actions = append(br.Actions, b.settleRound()...)

		result.Rounds = append(result.Rounds, br)

		// 回合结束判定：某一方大营有效兵力归零 → 该方败；双方均归零 → 同归于尽
		b.checkFinish()
	}

	// 平局：8 回合结束双方大营均有兵
	if !b.finish {
		b.draw = true
	}

	// 战后写回伤亡到快照，计算击杀
	syncUnitsToPb(b.attacker, attackerSlots)
	syncUnitsToPb(b.defender, defenderSlots)
	result.Attacker.KilledSoldiers = totalDead(b.defender)
	result.Defender.KilledSoldiers = totalDead(b.attacker)

	return result
}

// prepSide 战前准备：属性分层求和 + 技能加载
func prepSide(slots []*pb_battle.TeamSlotInfo, side int) []*battleUnit {
	units := make([]*battleUnit, 0, len(slots))
	for _, s := range slots {
		if s == nil || s.GetHeroInfo() == nil {
			continue
		}
		// 兵力状态：SoldierInfo 为 nil 时按 0 兵力处理（防御方无队伍快照等兜底）
		var alive, injured uint32
		if si := s.GetHeroInfo().GetSoldierInfo(); si != nil {
			alive = si.GetCurAliveNum()
			injured = si.GetCurInjuredNum()
		}
		units = append(units, &battleUnit{
			slot:        s.GetSlotId(),
			side:        side,
			heroID:      s.GetHeroInfo().GetHeroId(),
			level:       s.GetHeroInfo().GetCurLevel(),
			attrs:       slotAttr(s),
			attackRange: s.GetHeroInfo().GetAttackRange(),
			alive:       alive,
			injured:     injured,
			skills:      s.GetHeroInfo().GetSkills(),
		})
	}
	return units
}

// syncUnitsToPb 战后把兵力状态写回 pb 快照
func syncUnitsToPb(units []*battleUnit, slots []*pb_battle.TeamSlotInfo) {
	bySlot := make(map[int32]*pb_battle.TeamSlotInfo, len(slots))
	for _, s := range slots {
		if s != nil {
			bySlot[s.GetSlotId()] = s
		}
	}
	for _, u := range units {
		if pb := bySlot[u.slot]; pb != nil {
			if pb.HeroInfo == nil {
				pb.HeroInfo = &pb_hero.HeroInfo{}
			}
			if pb.HeroInfo.SoldierInfo == nil {
				pb.HeroInfo.SoldierInfo = &pb_hero.SoldierInfo{}
			}
			pb.HeroInfo.SoldierInfo.CurAliveNum = u.alive
			pb.HeroInfo.SoldierInfo.CurInjuredNum = u.injured
		}
	}
}

// actionOrder 所有存活英雄按移动(速度)降序；同速按槽位升序
func (b *battleCtx) actionOrder() []*battleUnit {
	var all []*battleUnit
	all = append(all, b.attacker...)
	all = append(all, b.defender...)
	alive := make([]*battleUnit, 0, len(all))
	for _, u := range all {
		if u.alive > 0 {
			alive = append(alive, u)
		}
	}
	for i := 1; i < len(alive); i++ {
		for j := i; j > 0; j-- {
			a, c := alive[j-1], alive[j]
			if a.attrs.Movement < c.attrs.Movement ||
				(a.attrs.Movement == c.attrs.Movement && a.slot > c.slot) {
				alive[j-1], alive[j] = c, a
			}
		}
	}
	return alive
}

// resolveUnitAction 单个英雄行动：主动技能 → 普攻 → 追击技能
func (b *battleCtx) resolveUnitAction(actor *battleUnit) []*pb_battle.BattleAction {
	var actions []*pb_battle.BattleAction

	// 主动技能
	if sc := findSkillConf(actor, skill.SkillTypeActive); sc != nil {
		if act := b.execSkill(actor, sc, pb_battle.BattleActionType_BATTLE_ACTION_ACTIVE_SKILL); act != nil {
			actions = append(actions, act)
		}
	}

	// 普攻（存活才行动）
	if actor.alive > 0 {
		if act := b.normalAttack(actor); act != nil {
			actions = append(actions, act)
		}
	}

	// 追击技能（概率触发）
	if sc := findSkillConf(actor, skill.SkillTypePursuit); sc != nil && roll(sc.TriggerRate) {
		if act := b.execSkill(actor, sc, pb_battle.BattleActionType_BATTLE_ACTION_PURSUIT); act != nil {
			actions = append(actions, act)
		}
	}

	return actions
}

// normalAttack 普攻：攻击范围内随机单个敌方对象，物理伤害
func (b *battleCtx) normalAttack(actor *battleUnit) *pb_battle.BattleAction {
	target := b.pickTarget(actor, skill.TargetRandom)
	if target == nil {
		return nil
	}
	dmg, inj, killed := b.applyDamage(actor, target, b.rules.PhysCoeff(), 100)
	return b.actionLog(actor, target, pb_battle.BattleActionType_BATTLE_ACTION_NORMAL_ATTACK, 0, dmg, inj, killed, 0)
}

// execSkill 执行技能（主动/追击）：按目标类型选目标，按效果类型算伤害/恢复
func (b *battleCtx) execSkill(actor *battleUnit, sc *skill.SkillConf, at pb_battle.BattleActionType) *pb_battle.BattleAction {
	target := b.pickTarget(actor, sc.TargetType)
	if target == nil {
		return nil
	}
	switch sc.EffectType {
	case skill.EffectRecover:
		heal := recoverInjured(target, sc.DamageCoeff)
		return b.actionLog(actor, target, at, sc.ConfID, 0, 0, 0, heal)
	case skill.EffectMagicDamage:
		coeff := sc.ConvergeCoeff
		if coeff == 0 {
			coeff = b.rules.MagicConverge
		}
		dmg, inj, killed := b.applyMagicDamage(actor, target, float64(coeff)/100, sc.DamageCoeff)
		return b.actionLog(actor, target, at, sc.ConfID, dmg, inj, killed, 0)
	default: // EffectPhysDamage
		coeff := sc.ConvergeCoeff
		if coeff == 0 {
			coeff = b.rules.PhysConverge
		}
		dmg, inj, killed := b.applyDamage(actor, target, float64(coeff)/100, sc.DamageCoeff)
		return b.actionLog(actor, target, at, sc.ConfID, dmg, inj, killed, 0)
	}
}

// pickTarget 按目标类型从攻击范围内选敌方目标
func (b *battleCtx) pickTarget(actor *battleUnit, tt skill.TargetType) *battleUnit {
	reachable := b.reachableEnemies(actor)
	if len(reachable) == 0 {
		return nil
	}
	switch tt {
	case skill.TargetBase:
		for _, u := range reachable {
			if u.slot == 1 { // 大营
				return u
			}
		}
		return nil
	case skill.TargetFront:
		best := reachable[0]
		for _, u := range reachable[1:] {
			if stanceDistance(actor.side, actor.slot, u.side, u.slot) <
				stanceDistance(actor.side, actor.slot, best.side, best.slot) {
				best = u
			}
		}
		return best
	default: // TargetRandom / TargetNone
		b.step++
		return reachable[int(b.step)%len(reachable)]
	}
}

// reachableEnemies 攻击范围内（站位距离 ≤ actor.attackRange）的存活敌方
func (b *battleCtx) reachableEnemies(actor *battleUnit) []*battleUnit {
	enemies := b.attacker
	if actor.side == 0 {
		enemies = b.defender
	}
	out := make([]*battleUnit, 0, len(enemies))
	for _, u := range enemies {
		if u.alive == 0 {
			continue
		}
		if actor.attackRange > 0 &&
			stanceDistance(actor.side, actor.slot, u.side, u.slot) <= int32(actor.attackRange) {
			out = append(out, u)
		}
	}
	return out
}

// applyDamage 物理伤害：攻击²/(攻击 + 防御×系数) × 当前有效兵力 × (伤害系数%)
//
// 返回 (damage, injured, killed)，damage = injured + killed = 有效兵力损失
func (b *battleCtx) applyDamage(actor, target *battleUnit, convergeCoeff float64, damageCoeff uint32) (uint32, uint32, uint32) {
	dmg := convergeHit(actor.attrs.Attack, target.attrs.Defense, convergeCoeff, float64(damageCoeff)/100, actor.alive)
	return b.damageTarget(target, dmg)
}

// applyMagicDamage 法术伤害：智力²/(智力 + 智力×系数) × 当前有效兵力 × (伤害系数%)
func (b *battleCtx) applyMagicDamage(actor, target *battleUnit, convergeCoeff float64, damageCoeff uint32) (uint32, uint32, uint32) {
	dmg := convergeHit(actor.attrs.Intelligence, target.attrs.Intelligence, convergeCoeff, float64(damageCoeff)/100, actor.alive)
	return b.damageTarget(target, dmg)
}

// convergeHit 收敛伤害：base = atk²/(atk + def×coeff)；结果 = base × damageCoeff × alive
func convergeHit(atkVal, defVal uint32, coeff, damageCoeff float64, alive uint32) uint32 {
	a := float64(atkVal)
	if a == 0 {
		return 0
	}
	base := a * a / (a + float64(defVal)*coeff)
	return uint32(base * damageCoeff * float64(alive))
}

// damageTarget 对目标造成有效兵力损失，按受伤比例拆死亡+伤兵
func (b *battleCtx) damageTarget(target *battleUnit, dmg uint32) (uint32, uint32, uint32) {
	if dmg > target.alive {
		dmg = target.alive
	}
	injuryRate := b.rules.InjuryRate(b.round)
	dead := dmg * (100 - injuryRate) / 100
	injured := dmg - dead
	target.alive -= dmg
	target.injured += injured
	target.dead += dead
	return dmg, injured, dead
}

// recoverInjured 恢复伤兵（技能 EffectRecover），返回恢复量
func recoverInjured(target *battleUnit, recoverCoeff uint32) uint32 {
	if recoverCoeff == 0 {
		recoverCoeff = 100
	}
	heal := uint32(float64(target.injured) * float64(recoverCoeff) / 100)
	if heal > target.injured {
		heal = target.injured
	}
	target.injured -= heal
	target.alive += heal
	return heal
}

// settleRound 回合结算：当前伤兵 10% 转死亡兵
func (b *battleCtx) settleRound() []*pb_battle.BattleAction {
	var actions []*pb_battle.BattleAction
	for _, side := range [][]*battleUnit{b.attacker, b.defender} {
		for _, u := range side {
			if u.injured == 0 {
				continue
			}
			dead := u.injured * b.rules.SettlementDeadRate / 100
			if dead == 0 {
				continue
			}
			u.injured -= dead
			u.dead += dead
			actions = append(actions, &pb_battle.BattleAction{
				ActorSlot:   u.slot,
				ActionType:  pb_battle.BattleActionType_BATTLE_ACTION_SETTLEMENT,
				Killed:      dead,
				TargetSlot:  u.slot,
			})
		}
	}
	return actions
}

// actionLog 构造行动日志
func (b *battleCtx) actionLog(actor, target *battleUnit, at pb_battle.BattleActionType, skillID int32,
	dmg, inj, killed, heal uint32) *pb_battle.BattleAction {
	return &pb_battle.BattleAction{
		ActorSlot:   actor.slot,
		ActionType:  at,
		SkillConfId: skillID,
		TargetSlot:  target.slot,
		Damage:      dmg,
		Injured:     inj,
		Killed:      killed,
		Heal:        heal,
	}
}

// checkFinish 大营判定：某一方大营有效兵力==0 → 该方败；双方同时 → 同归于尽
func (b *battleCtx) checkFinish() {
	attDead := baseDead(b.attacker)
	defDead := baseDead(b.defender)
	if attDead && defDead {
		b.finish = true
		b.draw = false // 同归于尽，非平局
		return
	}
	if attDead || defDead {
		b.finish = true
	}
}

// baseDead 该方大营(slot1)有效兵力是否归零（无大营按败）
func baseDead(units []*battleUnit) bool {
	for _, u := range units {
		if u.slot == 1 { // 大营
			return u.alive == 0
		}
	}
	return true
}

// totalDead 该方累计死亡兵数
func totalDead(units []*battleUnit) uint64 {
	var sum uint64
	for _, u := range units {
		sum += uint64(u.dead)
	}
	return sum
}

// findSkillConf 找英雄身上指定类型的第一个技能（查技能表）
func findSkillConf(actor *battleUnit, typ skill.SkillType) *skill.SkillConf {
	for _, s := range actor.skills {
		if s == nil {
			continue
		}
		sc, ok := game_conf.Load().Skill.GetSkillConf(s.GetConfigId())
		if !ok {
			continue
		}
		if sc.SkillType == typ {
			return &sc
		}
	}
	return nil
}

// roll 概率判定（percent）— 确定性伪随机（全局步进），测试可复现
func roll(rate uint32) bool {
	if rate == 0 {
		return false
	}
	if rate >= 100 {
		return true
	}
	return (stepCounter()*7+3)%100 < rate
}

// stepCounter 全局步进（追击概率判定的确定性伪随机源）
var _stepCounter uint32

func stepCounter() uint32 {
	_stepCounter++
	if _stepCounter > 1000000 {
		_stepCounter = 0
	}
	return _stepCounter
}
