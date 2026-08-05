package game_logics

import (
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
)

// heroCardConsumable 英雄卡是否可被消耗（升星消耗等）
//
// 不可消耗：被锁定、被编队引用、有技能装配（EquipSkills 非空）、有养成投入（升过星/升过级）。
func heroCardConsumable(role *game_roles.Role, card *role_heroes.RoleHero) bool {
	if card.GetIsLocked() {
		return false
	}
	if role.GetFormations().FormationHasHero(card.GetID()) {
		return false
	}
	if len(card.EquipSkills) > 0 {
		return false
	}
	if card.GetStarStage() > 0 || card.GetLevel() > 1 {
		return false
	}
	return true
}

// HeroUpgradeStar 英雄升星：消耗一张同配置英雄卡，升 1 星
//
//   - 满星 → HeroStarFull
//   - 无同配置其他卡 → HeroNoConsumeCard
//   - 消耗卡从内存与 DB 移除，被消耗卡的配置记录进养成消耗记录（CultivateStar）
func HeroUpgradeStar(role *game_roles.Role, hero *role_heroes.RoleHero) error {
	if hero.GetStarStage() >= game_conf.Load().Hero.MaxStarStage {
		return rpc_results.Error(pb_error_code.ErrorCode_HeroStarFull, "hero already max star stage")
	}

	// 找同配置的其他英雄卡（排除自身；跳过被编队/技能装配/养成引用、不可消耗的卡；后续可扩展为兼容英雄卡）
	var consumeCard *role_heroes.RoleHero
	for _, card := range role.GetHeroes().GetHeroesByConf(hero.GetHeroConfID()) {
		if card.GetID() == hero.GetID() {
			continue
		}
		if heroCardConsumable(role, card) {
			consumeCard = card
			break
		}
	}
	if consumeCard == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_HeroNoConsumeCard, "no same config hero card to consume")
	}

	// 消耗卡：内存移除 + DB 删除（无 writeDB 时（如单测）跳过 DB，仅内存）
	role.GetHeroes().RemoveHero(consumeCard.GetID())
	if db := dbconn.GetWriteDbConn(); db != nil {
		if err := role.GetHeroes().DBDeleteHero(db, consumeCard.GetID()); err != nil {
			return err
		}
	}

	// 升星
	hero.SetStarStage(hero.GetStarStage() + 1)

	// 记录养成消耗：被消耗英雄卡配置
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateStar, []*pb_common.Int32KV{
		{Key: hero.GetHeroConfID(), Val: 1},
	})

	return nil
}
