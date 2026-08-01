package march_consumer

import (
	"context"
	"time"

	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const marchStreamKey = "slg:march:events"

var lastID string // XREAD 游标

// Init 启动行军事件消费者协程
func Init(parentCtx context.Context) {
	lastID = "0" // 从 stream 头部开始读取
	go consumeLoop(parentCtx)
}

func consumeLoop(ctx context.Context) {
	loggers.Logger.Info("march consumer started")

	for {
		select {
		case <-ctx.Done():
			loggers.Logger.Info("march consumer stopped")
			return
		default:
		}

		readEvents(ctx)
	}
}

// readEvents 从 Redis Stream 阻塞读取事件
func readEvents(ctx context.Context) {
	cache := cacheconn.Get()

	type streamReadI interface {
		XRead(ctx context.Context, a *redis.XReadArgs) *redis.XStreamSliceCmd
	}
	reader, ok := cache.(streamReadI)
	if !ok {
		loggers.Logger.Warn("cache does not support XREAD, retry in 5s")
		time.Sleep(5 * time.Second)
		return
	}

	readCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	streams, err := reader.XRead(readCtx, &redis.XReadArgs{
		Streams: []string{marchStreamKey, lastID},
		Count:   10,
		Block:   8 * time.Second,
	}).Result()

	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return
		}
		loggers.Logger.Warn("XRead error", zap.Error(err))
		time.Sleep(time.Second)
		return
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			lastID = msg.ID
			handleMessage(ctx, msg.Values)
		}
	}
}

// handleMessage 处理单条 Redis Stream 消息
func handleMessage(_ context.Context, values map[string]interface{}) {
	dataStr, ok := values["data"].(string)
	if !ok {
		loggers.Logger.Warn("march event missing 'data' field")
		return
	}

	var ev pb_redis_stream.MarchEvent
	if err := proto.Unmarshal([]byte(dataStr), &ev); err != nil {
		loggers.Logger.Warn("march event unmarshal failed", zap.Error(err))
		return
	}

	switch ev.Type {
	case pb_redis_stream.MarchEventType_MARCH_EVENT_ARRIVED:
		loggers.Logger.Info("march arrived event",
			zap.Uint64("march_id", ev.MarchId),
			zap.Uint64("role_id", ev.RoleId),
			zap.Int32("to_map_id", ev.ToMapId),
			zap.Int32("march_type", ev.MarchType),
			zap.Int32("state", ev.State))

		switch pb_maps_march.MarchState(ev.State) {
		case pb_maps_march.MarchState_Stay:
			// 到达并驻留（战斗胜利 / 占领成功 / 驻守到达）
			// TODO: 处理驻留后的业务（占领奖励、驻守生效等）
		case pb_maps_march.MarchState_Back:
			// 结算失败回城，或召回到达
			// TODO: 处理回城（归还士兵、解锁队伍等）
		default:
			// 其他状态（采集、扫荡等后续 march type 扩展）
			// TODO: 处理采集开始、扫荡结算等
		}

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
}
