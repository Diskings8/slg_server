package stream_consumers

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"

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

	case pb_redis_stream.MarchEventType_MARCH_EVENT_BACKARRIVED:
		loggers.Logger.Info("march back arrived event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId))
		// TODO: 处理行军回城到站（归还士兵、解锁队伍等）

	case pb_redis_stream.MarchEventType_MARCH_EVENT_CANCELED:
		loggers.Logger.Info("march canceled event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId))
		// TODO: 处理行军取消（释放队伍、归还士兵等）

	default:
		loggers.Logger.Warn("unknown march event type",
			zap.String("type", ev.Type.String()),
			zap.Uint64("march_id", ev.MarchId))
	}

	return nil
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
