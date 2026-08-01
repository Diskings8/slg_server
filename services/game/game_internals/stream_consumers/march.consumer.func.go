package stream_consumers

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"

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
		// TODO: 处理行军到达（战斗结算、采集开始等）

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
