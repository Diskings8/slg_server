package game_logics

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// HeroAwaken 英雄觉醒：解锁第三技能槽（heroSkillSlotUnlocked 的 slot=2 判定）
//
//   - 前置：等级 ≥ hero.awaken_level；未觉醒
//   - 消耗 hero.awaken_cost（资源混搭，走 ItemChange 统一扣除）
//   - 觉醒后 IsAwakened=true，记录养成消耗
func HeroAwaken(role *game_roles.Role, hero *role_heroes.RoleHero) error {
	if hero.GetIsAwakened() {
		return rpc_results.Error(pb_error_code.ErrorCode_HeroAlreadyAwakened, "hero already awakened")
	}
	hc := game_conf.Load().Hero
	if hero.GetLevel() < hc.AwakenLevel {
		return rpc_results.Error(pb_error_code.ErrorCode_HeroAwakenLevelNotEnough, "hero level below awaken requirement")
	}

	if err := ItemChange(role, nil, hc.AwakenCost, common_declarations.ReasonSkill); err != nil {
		return err
	}

	// 记录养成消耗（本次觉醒消耗的资源）
	costs := make([]*pb_common.Int32KV, 0, len(hc.AwakenCost))
	for _, c := range hc.AwakenCost {
		costs = append(costs, &pb_common.Int32KV{Key: int32(c.ItemID), Val: int32(c.Count)})
	}
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateAwaken, costs)

	hero.SetIsAwakened(true)
	return nil
}
