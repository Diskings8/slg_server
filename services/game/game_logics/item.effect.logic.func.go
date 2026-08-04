package game_logics

import (
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// ApplyItemEffect 使用道具：扣道具 + 按配置执行效果
//
//   - 前置：道具配置存在（含 effect）
//   - 消耗不足时返回携带 ItemTypeNormalNotEnough 的 error
//   - targetHeroID：EffectAddHeroExp 目标英雄（0 或无需目标的 effect 忽略）
func ApplyItemEffect(role *game_roles.Role, configID pb_confs.ItemID, count int64, targetHeroID int64) error {
	conf, ok := game_conf.Load().Item.Get(configID)
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_ItemEffectConfNotFound, fmt.Sprintf("item effect conf not found: %d", configID))
	}

	// 前置校验（扣道具前，避免扣了道具效果失败丢道具）
	var targetHero *role_heroes.RoleHero
	if conf.Effect.Type == item.EffectAddHeroExp {
		targetHero = role.GetHeroes().GetHero(pb_confs.ItemID(targetHeroID))
		if targetHero == nil {
			return rpc_results.Error(pb_error_code.ErrorCode_ItemEffectTargetInvalid, "item effect target hero not found")
		}
	}

	// 统一扣道具
	if err := ItemChange(role, nil, []common_declarations.ItemUse{
		{ItemID: configID, Count: count},
	}, common_declarations.ReasonUse); err != nil {
		return err
	}

	// 按效果分发
	switch conf.Effect.Type {
	case item.EffectAddHeroExp:
		_, err := HeroAddExp(targetHero, uint32(conf.Effect.Value)*uint32(count))
		return err
	case item.EffectAddCurrency:
		return ItemChange(role, []common_declarations.ItemUse{
			{ItemType: pb_confs.ItemTypeCurrency2, ItemID: pb_confs.ItemID(conf.Effect.Target), Count: conf.Effect.Value * count},
		}, nil, common_declarations.ReasonReward)
	case item.EffectAddItem:
		return ItemChange(role, []common_declarations.ItemUse{
			{ItemType: pb_confs.ItemTypeNormal, ItemID: pb_confs.ItemID(conf.Effect.Target), Count: conf.Effect.Value * count},
		}, nil, common_declarations.ReasonReward)
	}
	// EffectNone：仅消耗，无附加效果
	return nil
}
