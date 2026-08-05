package game_logics

import (
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// SkillCollectionActivate 技能收藏激活（消耗客户端选定的一张英雄卡）
//
// 消耗传入的英雄卡 → CollectionLevel 累积进度 → 全部达标解锁对应技能并发放到技能库。
//
//   - 该卡英雄配置必须是该收藏配置所需；该英雄配置已收集满时不可再消耗
//   - 该卡不可消耗（锁定/编队/装配技能/养成）时返回携带 HeroNoConsumeCard 的 error
func SkillCollectionActivate(role *game_roles.Role, skillConfID int32, heroID uint64) error {
	conf, ok := game_conf.Load().Skill.GetCollectionConf(skillConfID)
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionConfNotFound, fmt.Sprintf("skill collection conf not found: %d", skillConfID))
	}

	// 客户端选定的英雄卡
	hero := role.GetHeroes().GetHero(heroID)
	if hero == nil {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "hero card not found")
	}

	// 校验该卡英雄配置是收藏所需
	var need int64
	isNeed := false
	for _, needHero := range conf.NeedHeroes {
		if int32(needHero.ItemID) == hero.GetHeroConfID() {
			need = needHero.Count
			isNeed = true
			break
		}
	}
	if !isNeed {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionHeroInvalid, fmt.Sprintf("hero not required by skill collection: %d", hero.GetHeroConfID()))
	}

	collection := role.GetSkillCollections().GetBySkillConfID(skillConfID)
	if collection == nil {
		collection = role.GetSkillCollections().AddSkillCollection(skillConfID)
	}
	if collection.IsUnlocked {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionUnlocked, "skill collection already unlocked")
	}

	// 该英雄配置已收集满 → 不可再消耗
	if collectionCollected(collection.CollectionLevel, hero.GetHeroConfID()) >= need {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionHeroFull, fmt.Sprintf("hero already fully collected: %d", hero.GetHeroConfID()))
	}

	// 可消耗守卫：未锁定/未编队/无装配技能/未养成（防止误消耗已投入的英雄）
	if !heroCardConsumable(role, hero) {
		return rpc_results.Error(pb_error_code.ErrorCode_HeroNoConsumeCard, "hero card not consumable")
	}

	// 消耗该英雄卡：内存移除 + DB 删除（无 writeDB 时（如单测）跳过 DB，仅内存）
	role.GetHeroes().RemoveHero(hero.GetID())
	if writeDB := dbconn.GetWriteDbConn(); writeDB != nil {
		if err := role.GetHeroes().DBDeleteHero(writeDB, hero.GetID()); err != nil {
			return err
		}
	}

	// 记录养成消耗（消耗的英雄配置 + 数量1）
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateSkill, []*pb_common.Int32KV{
		{Key: hero.GetHeroConfID(), Val: 1},
	})

	// 累积收集进度
	collection.CollectionLevel = addCollectionProgress(collection.CollectionLevel, hero.GetHeroConfID(), 1)

	// 全部所需英雄达标 → 解锁并发放技能到技能库（可装配）
	if collectionComplete(collection.CollectionLevel, conf.NeedHeroes) {
		collection.IsUnlocked = true
		if skillConf, ok := game_conf.Load().Skill.GetSkillConf(skillConfID); ok {
			role.GetSkills().UnlockSkill(skillConfID, skillConf.UseLimit)
		}
	}
	return nil
}

// collectionCollected 获取某道具已收集数量
func collectionCollected(level []*pb_common.Int32KV, itemConfID int32) int64 {
	for _, kv := range level {
		if kv.GetKey() == itemConfID {
			return int64(kv.GetVal())
		}
	}
	return 0
}

// addCollectionProgress 累积收集进度（某道具已收集数量 + count）
func addCollectionProgress(level []*pb_common.Int32KV, itemConfID int32, count int64) []*pb_common.Int32KV {
	for _, kv := range level {
		if kv.GetKey() == itemConfID {
			kv.Val = kv.GetVal() + int32(count)
			return level
		}
	}
	return append(level, &pb_common.Int32KV{Key: itemConfID, Val: int32(count)})
}

// collectionComplete 所有所需道具是否已达标
func collectionComplete(level []*pb_common.Int32KV, needItems []common_declarations.ItemUse) bool {
	for _, needItem := range needItems {
		if collectionCollected(level, int32(needItem.ItemID)) < needItem.Count {
			return false
		}
	}
	return true
}
