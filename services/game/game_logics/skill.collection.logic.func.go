package game_logics

import (
	"fmt"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
)

// SkillCollectionActivate 技能收藏激活（分次收集）
//
// 消耗配置所需道具 count 个 → CollectionLevel 累积进度 → 全部达标解锁对应技能。
//
//   - itemConfID 必须是该收藏配置所需道具；该道具已收集满时不可再消耗
//   - 消耗不足返回携带 ItemTypeNormalNotEnough 的 error
func SkillCollectionActivate(role *game_roles.Role, skillConfID, itemConfID int32, count int64) error {
	conf, ok := game_conf.Load().Skill.GetCollectionConf(skillConfID)
	if !ok {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionConfNotFound, fmt.Sprintf("skill collection conf not found: %d", skillConfID))
	}

	// 校验消耗的道具是配置所需
	var need int64
	isNeed := false
	for _, needItem := range conf.NeedItems {
		if int32(needItem.ItemID) == itemConfID {
			need = needItem.Count
			isNeed = true
			break
		}
	}
	if !isNeed {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionItemInvalid, fmt.Sprintf("item not required by skill collection: %d", itemConfID))
	}

	collection := role.GetSkillCollections().GetBySkillConfID(skillConfID)
	if collection == nil {
		collection = role.GetSkillCollections().AddSkillCollection(skillConfID)
	}
	if collection.IsUnlocked {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionUnlocked, "skill collection already unlocked")
	}

	// 该道具已收集满 → 不可再消耗
	if collectionCollected(collection.CollectionLevel, itemConfID) >= need {
		return rpc_results.Error(pb_error_code.ErrorCode_SkillCollectionItemFull, fmt.Sprintf("item already fully collected: %d", itemConfID))
	}

	// 扣道具（配置中该道具为普通道具）
	if err := ItemChange(role, nil, []common_declarations.ItemUse{
		{ItemID: pb_confs.ItemID(itemConfID), Count: count},
	}, common_declarations.ReasonUse); err != nil {
		return err
	}

	// 记录养成消耗（本次消耗的道具 config + 数量）
	role.GetCultivateCosts().AddCost(pb_cultivate.CultivateType_CultivateSkill, []*pb_common.Int32KV{
		{Key: itemConfID, Val: int32(count)},
	})

	// 累积收集进度
	collection.CollectionLevel = addCollectionProgress(collection.CollectionLevel, itemConfID, count)

	// 全部所需道具达标 → 解锁
	if collectionComplete(collection.CollectionLevel, conf.NeedItems) {
		collection.IsUnlocked = true
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
