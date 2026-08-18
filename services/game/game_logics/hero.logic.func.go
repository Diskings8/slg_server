package game_logics

import (
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// HeroAddExp 英雄获得经验 → 升级判断（服务端内部共用入口，不对前端直接开放）
//
// exp 来源仅两个：
//
//  1. 道具 effect（经验书）→ ApplyItemEffect 已扣道具后调用
//
//  2. 战斗结算经验 → 战斗回调接入后由服务端调用
//
//     - 累加经验后循环判断：满足当前等级所需经验（配置 hero.NeedExp）即升级（可能连升多级）
//     - 每升 10 级发放自由属性点（配置 hero.FreePointPer10L）
//     - 达到等级上限后多余经验不再触发升级
//
//     - 返回: 升级后的等级
func HeroAddExp(hero *role_heroes.RoleHero, exp uint32) (uint32, error) {
	if exp == 0 {
		return hero.GetLevel(), nil
	}

	hc := game_conf.Load().Hero
	newExp := uint64(hero.GetExp()) + uint64(exp)
	level := hero.GetLevel()

	for level < hc.MaxLevel {
		need := uint64(hc.NeedExp(level))
		if newExp < need {
			break
		}
		newExp -= need
		level++

		// 每10级获得自由属性点
		if level%10 == 0 {
			hero.SetAttrPoint(hero.GetAttrPoint() + hc.FreePointPer10L)
		}
	}

	hero.SetLevel(level)
	hero.SetExp(uint32(newExp))
	hero.RefreshCurVal() // 升级后按新等级重算 cur_val（battle 快照用）

	return level, nil
}

// GmSetHeroLevel GM：设置英雄等级（传高于当前值即提升），经验清零并重算战斗属性。
// 供测试/运营调试用，非玩家路径。
func GmSetHeroLevel(hero *role_heroes.RoleHero, level uint32) error {
	if level < 1 || level > game_conf.Load().Hero.MaxLevel {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "level out of range")
	}
	hero.SetLevel(level)
	hero.SetExp(0)
	hero.RefreshCurVal() // 按新等级重算 cur_val（battle 快照用）
	return nil
}

// HeroCultivate 英雄加点
//
//   - 消耗 1 点自由属性点，给指定属性 +1（add_val_camp）
//   - cultivateType: 0=攻击 1=防御 2=智力 3=移动（拆迁值 4 不可加点）
//   - 返回: 加点后的属性
func HeroCultivate(hero *role_heroes.RoleHero, cultivateType uint32) (*pb_cultivate.Cultivate, error) {
	if cultivateType > 3 {
		return nil, fmt.Errorf("cultivate type %d not allowed, relocation cannot add point", cultivateType)
	}
	if hero.GetAttrPoint() < 1 {
		return nil, fmt.Errorf("no attr point left")
	}

	// 补齐5维数组
	cultivates := hero.Cultivates
	for len(cultivates) < 5 {
		cultivates = append(cultivates, &pb_cultivate.Cultivate{})
	}

	attr := cultivates[cultivateType]
	attr.AddValCamp++
	hero.Cultivates = cultivates

	// 消耗自由属性点
	hero.SetAttrPoint(hero.GetAttrPoint() - 1)

	return attr, nil
}
