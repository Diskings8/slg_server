package stream_consumers

import (
	"context"

	"server.slg.com/common/redisstream"
)

// Init 启动行军事件消费者协程
//
// 在此注册需要监听的key和回调
func Init(parentCtx context.Context) {
	redisstream.MultiConsume(parentCtx, map[string]redisstream.Handler{
		redisstream.StreamKeyMarchEvents: handleMessage,
	}, nil)
}
