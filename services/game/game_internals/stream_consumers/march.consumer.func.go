package stream_consumers

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
	"server.slg.com/services/internal/cores/cores_declarations"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// handleMessage 处理单条 Redis Stream 消息
func handleMessage(_ context.Context, msg redisstream.Message) error {
	dataStr, ok := msg.Values["data"].(string)
	if !ok {
		loggers.Logger.Warn("march event missing 'data' field")
		return nil
	}

	var ev pb_redis_stream.MarchEvent
	if err := proto.Unmarshal([]byte(dataStr), &ev); err != nil {
		loggers.Logger.Warn("march event unmarshal failed", zap.Error(err))
		return nil
	}

	switch ev.Type {
	case pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED:
		loggers.Logger.Info("march arrived event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId),
			zap.Int32("to_map_id", ev.ToMapId),
			zap.Int32("march_type", ev.MarchType))
		// 战斗经验发放：攻击行军 ARRIVED 携带 battle_result（每英雄总经验）
		handleBattleExp(&ev)
		// 资源地产出快照同步：开发升级 / 攻占得手 → 更新地块产出
		handleTileSync(&ev)

	case pb_redis_stream.MarchEventType_MARCH_EVENT_BACKARRIVED:
		loggers.Logger.Info("march back arrived event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId))
		// 回城战损写回：把行军战后存活兵力写回 formation（不释放、不恢复，补兵机制后续）
		handleMarchBackArrived(&ev)

	case pb_redis_stream.MarchEventType_MARCH_EVENT_CANCELED:
		loggers.Logger.Info("march canceled event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId))
		// TODO: 处理行军取消（释放队伍、归还士兵等）

	case pb_redis_stream.MarchEventType_MARCH_EVENT_TILE_RELEASED:
		// 地块被放弃：移除资源地快照（停止产出）
		syncRemoveTile(ev.GetRoleId(), ev.GetToMapId())

	default:
		loggers.Logger.Warn("unknown march event type",
			zap.String("type", ev.Type.String()),
			zap.Uint64("march_id", ev.MarchId))
	}

	return nil
}

// handleMarchBackArrived 回城到站战损写回：消费 MarchEvent.team_info 的战后存活兵力，
// 按 hero_id 匹配角色全部 formation 槽位并写回 SoldierNum（保留战损值，不补满）。
// 补兵机制（消耗资源补到上限）后续接入。
func handleMarchBackArrived(ev *pb_redis_stream.MarchEvent) {
	team := ev.GetTeamInfo()
	if team == nil || len(team.GetSlotInfo()) == 0 {
		return
	}

	// 战后存活兵力：hero_id → cur_alive_num
	alive := make(map[uint64]uint32, len(team.GetSlotInfo()))
	for _, s := range team.GetSlotInfo() {
		if s == nil || s.GetHeroInfo() == nil || s.GetHeroInfo().GetSoldierInfo() == nil {
			continue
		}
		alive[s.GetHeroInfo().GetHeroId()] = s.GetHeroInfo().GetSoldierInfo().GetCurAliveNum()
	}
	if len(alive) == 0 {
		return
	}

	poller, role, err := game_roles.GetRole(ev.GetRoleId())
	if err != nil {
		return
	}
	defer poller.Release()

	changed := false
	for _, f := range role.GetFormations().List {
		for _, hs := range f.HeroSlots {
			if hs == nil {
				continue
			}
			if cur, ok := alive[hs.GetHeroId()]; ok {
				hs.SoldierNum = cur
				changed = true
			}
		}
	}
	if changed {
		poller.Save()
	}
}

// handleBattleExp 战斗经验发放：消费 MarchEvent.battle_result，给参战英雄累加经验（HeroAddExp 内部判升级）。
//
// 英雄不存在（非本服/已删除）跳过；至少发放成功一个才 Save 打脏。
func handleBattleExp(ev *pb_redis_stream.MarchEvent) {
	result := ev.GetBattleResult()
	if result == nil || len(result.GetHeroExp()) == 0 {
		return
	}

	poller, role, err := game_roles.GetRole(ev.GetRoleId())
	if err != nil {
		return
	}
	defer poller.Release()

	granted := false
	for _, item := range result.GetHeroExp() {
		hero := role.GetHeroes().GetHero(item.GetHeroId())
		if hero == nil {
			continue
		}
		if _, lerr := game_logics.HeroAddExp(hero, item.GetExp()); lerr == nil {
			granted = true
		}
	}
	if granted {
		poller.Save()
	}
}

// handleTileSync 目标地块产出快照同步：
//   - 开发(10005) 成功 → 快照升级（SyncResourceTile 内部先按旧等级结算再更新）
//   - 攻占(10001) 胜利 → 快照新增/更新；目标非资源元素 → 移除
//   - 回城/取消/其他行军 → 不改地块
func handleTileSync(ev *pb_redis_stream.MarchEvent) {
	if ev.GetState() == int32(pb_maps_march.MarchState_Back) {
		return
	}
	mt := cores_declarations.MarchType(ev.GetMarchType())
	switch mt {
	case cores_declarations.MarchTypeDevelop, cores_declarations.MarchTypeAttack:
		if mt == cores_declarations.MarchTypeAttack && !ev.GetBattleResult().GetAttackerWin() {
			return // 攻占失败不改归属
		}
		if ev.GetToMapId() <= 0 {
			return
		}
		// 攻占夺地：清理原归属者的资源地快照（被掠夺）
		if prev := ev.GetPrevOwner(); prev > 0 && prev != ev.GetRoleId() {
			syncRemoveTile(prev, ev.GetToMapId())
		}
		// 目标非资源元素（地形）→ 移除快照，不再产出
		if ev.GetTileElement() < int32(cores_declarations.ElementType_Resources_1) ||
			ev.GetTileElement() > int32(cores_declarations.ElementType_Resources_4) {
			syncRemoveTile(ev.GetRoleId(), ev.GetToMapId())
			return
		}
		syncUpsertTile(ev.GetRoleId(), ev.GetToMapId(), ev.GetTileLevel(), ev.GetTileElement())
	}
}

// syncUpsertTile 更新资源地快照（先结算旧状态再更新，见 game_logics.SyncResourceTile）
func syncUpsertTile(roleID uint64, mapID int32, level, element int32) {
	if roleID == 0 {
		return
	}
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return
	}
	defer poller.Release()
	game_logics.SyncResourceTile(role, roleID, mapID, level, element)
	poller.Save()
}

// syncRemoveTile 移除资源地快照（放弃/被夺/变为非资源）
func syncRemoveTile(roleID uint64, mapID int32) {
	if roleID == 0 {
		return
	}
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return
	}
	defer poller.Release()
	if role.GetResourceTiles().Remove(mapID) {
		poller.Save()
	}
}
