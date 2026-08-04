package game_logics

import (
	"errors"
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_equip"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// isBaseTroop 判断是否基础兵种（100/200/300 百位整百），否则为派生类型（101/102/103）
//
// TODO: 由配置表定义基础/派生关系，当前用命名规则占位
func isBaseTroop(troopTypeID int32) bool {
	return troopTypeID > 0 && troopTypeID%100 == 0
}

// ensureBaseTroop 确保英雄已有兵种列表（含自带基础类型）
func ensureBaseTroop(hero *role_heroes.RoleHero) {
	if len(hero.Troops) == 0 {
		hero.Troops = []*pb_equip.Troop{
			{ConfigId: game_conf.Load().Troop.DefaultTroopID, IsActivate: true},
		}
		hero.CurTroopTypeID = game_conf.Load().Troop.DefaultTroopID
	}
	if hero.CurTroopTypeID == 0 {
		hero.CurTroopTypeID = hero.Troops[0].GetConfigId()
	}
}

// isTroopUnlocked 判断兵种是否已解锁（在 Troops 中且激活）
func isTroopUnlocked(hero *role_heroes.RoleHero, troopTypeID int32) bool {
	for _, t := range hero.Troops {
		if t.GetConfigId() == troopTypeID && t.GetIsActivate() {
			return true
		}
	}
	return false
}

// HeroTroopUnlock 扩展兵种：使用道具解锁新的可选派生类型
//
//   - 基础类型自带，无需解锁
//   - 消耗扩展道具（配置：兵种 → 道具映射，当前占位固定道具）
func HeroTroopUnlock(role *game_roles.Role, hero *role_heroes.RoleHero, troopTypeID int32) error {
	ensureBaseTroop(hero)

	if isBaseTroop(troopTypeID) {
		return errors.New("base troop does not need unlock")
	}
	if isTroopUnlocked(hero, troopTypeID) {
		return errors.New("troop already unlocked")
	}

	// TODO 配置：兵种 → 所需扩展道具映射，当前占位固定道具
	cost := []common_declarations.ItemUse{
		{ItemID: pb_confs.ItemID(game_conf.Load().Troop.UnlockItemConf), Count: 1},
	}
	if err := ItemChange(role, nil, cost, common_declarations.ReasonTroop); err != nil {
		return err
	}

	// 记录养成消耗（解锁消耗的道具 config + 数量）
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateTroop, []*pb_common.Int32KV{
		{Key: game_conf.Load().Troop.UnlockItemConf, Val: 1},
	})

	// 已存在但未激活 → 激活；否则追加
	for _, t := range hero.Troops {
		if t.GetConfigId() == troopTypeID {
			t.IsActivate = true
			return nil
		}
	}
	hero.Troops = append(hero.Troops, &pb_equip.Troop{ConfigId: troopTypeID, IsActivate: true})
	return nil
}

// HeroTroopTransform 兵种转化：x 等级后选择已解锁派生类型（消耗资源）
//
//   - 需要等级 ≥ troopTransformLevel（配置）
//   - 目标必须是已解锁的派生类型
//   - 转化消耗资源（配置，当前占位不扣）
func HeroTroopTransform(hero *role_heroes.RoleHero, troopTypeID int32) error {
	ensureBaseTroop(hero)

	if hero.GetLevel() < game_conf.Load().Troop.TransformLevel {
		return fmt.Errorf("level %d below transform requirement %d", hero.GetLevel(), game_conf.Load().Troop.TransformLevel)
	}
	if isBaseTroop(troopTypeID) {
		return errors.New("cannot transform to base troop")
	}
	if !isTroopUnlocked(hero, troopTypeID) {
		return errors.New("troop not unlocked, use item to unlock first")
	}

	// TODO 配置：转化消耗资源（接入配置表后调用 ItemChange 扣除）
	hero.SetCurTroopTypeID(troopTypeID)
	return nil
}
