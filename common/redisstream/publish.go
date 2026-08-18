package redisstream

import (
	"context"
	"errors"
	"time"

	"server.slg.com/common/conns/cacheconn"

	"github.com/redis/go-redis/v9"
)

// ErrStreamNotSupported 缓存实例不支持 Stream 操作
var ErrStreamNotSupported = errors.New("cache instance does not support stream operations")

// streamMaxLen stream 近似裁剪长度（防止无限增长；消费游标持久化后旧消息不会再被重读）
const streamMaxLen = 1000

// streamXAdder Redis XADD 接口
type streamXAdder interface {
	XAdd(ctx context.Context, a *redis.XAddArgs) *redis.StringCmd
}

// ProtoXAdd 将 protobuf 序列化后的 bytes 发布到指定的 Redis Stream
// streamKey 为目标 stream 名称，data 为已序列化的 protobuf payload
// 内部使用 3s 超时的 context，避免阻塞过久
func ProtoXAdd(parentCtx context.Context, streamKey string, data []byte) error {
	cache := cacheconn.Get()
	pub, ok := cache.(streamXAdder)
	if !ok {
		return ErrStreamNotSupported
	}

	ctx, cancel := context.WithTimeout(parentCtx, 3*time.Second)
	defer cancel()

	return pub.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		MaxLen: streamMaxLen, // 近似裁剪旧消息，防止 stream 无限增长
		Approx: true,
		Values: []string{"data", string(data)},
	}).Err()
}
