package redisstream

import (
	"context"
	"errors"
	"time"

	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Message 封装一条 Redis Stream 消息
type Message struct {
	ID     string
	Values map[string]any
}

// Handler 用户自定义的消息处理函数
// 返回 error 时消费循环会打印警告日志并继续下一条
type Handler func(ctx context.Context, msg Message) error

// streamXReader Redis XREAD 接口
type streamXReader interface {
	XRead(ctx context.Context, a *redis.XReadArgs) *redis.XStreamSliceCmd
}

// ConsumeOpts 消费选项
type ConsumeOpts struct {
	Count     int64         // 每次读取的最大条数，默认 10
	BlockTime time.Duration // 阻塞超时，默认 8s
	Logger    *zap.Logger   // 日志器，默认使用 loggers.Logger
}

func (o *ConsumeOpts) fillDefaults() {
	if o.Count <= 0 {
		o.Count = 10
	}
	if o.BlockTime <= 0 {
		o.BlockTime = 8 * time.Second
	}
	if o.Logger == nil {
		o.Logger = loggers.Logger
	}
}

// readEvents 从 Redis Stream 阻塞读取一批消息
func readEvents(ctx context.Context, streamKey string, lastID string, opts *ConsumeOpts) ([]Message, error) {
	cache := cacheconn.Get()
	reader, ok := cache.(streamXReader)
	if !ok {
		return nil, ErrStreamNotSupported
	}

	readCtx, cancel := context.WithTimeout(ctx, opts.BlockTime+2*time.Second)
	defer cancel()

	streams, err := reader.XRead(readCtx, &redis.XReadArgs{
		Streams: []string{streamKey, lastID},
		Count:   opts.Count,
		Block:   opts.BlockTime,
	}).Result()
	if err != nil {
		return nil, err
	}

	var msgs []Message
	for _, stream := range streams {
		for _, msg := range stream.Messages {
			msgs = append(msgs, Message{
				ID:     msg.ID,
				Values: msg.Values,
			})
		}
	}
	return msgs, nil
}

// MultiConsume 并发启动多个 Stream 消费者，每个 key 一个独立协程
// 所有消费者共享相同的 opts（若指定）；透传 ctx，取消时全部退出
func MultiConsume(ctx context.Context, consumers map[string]Handler, opts *ConsumeOpts) {
	for streamKey, handler := range consumers {
		go consume(ctx, streamKey, handler, opts)
	}
}

// consume 启动阻塞式 Redis Stream 消费者
//
// 从 streamKey 持续读取消息，每批到达后逐一调用 handler 处理。
// 内部自动维护游标 lastID，初始从头开始读取。
// 当 ctx 取消时，循环正常退出。
//
// 错误处理策略：
//   - context.Canceled / DeadlineExceeded → 静默返回
//   - ErrStreamNotSupported → 打印警告并 sleep 5s 后重试
//   - 其他 XRead 错误 → sleep 1s 后重试
func consume(ctx context.Context, streamKey string, handler Handler, opts *ConsumeOpts) {
	if opts == nil {
		opts = &ConsumeOpts{}
	}
	opts.fillDefaults()

	log := opts.Logger
	lastID := "0"

	log.Info("redis stream consumer started", zap.String("stream", streamKey))

	for {
		select {
		case <-ctx.Done():
			log.Info("redis stream consumer stopped", zap.String("stream", streamKey))
			return
		default:
		}

		msgs, err := readEvents(ctx, streamKey, lastID, opts)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if errors.Is(err, ErrStreamNotSupported) {
				log.Warn("cache does not support XREAD, retry in 5s",
					zap.String("stream", streamKey))
				time.Sleep(5 * time.Second)
				continue
			}

			time.Sleep(time.Second)
			continue
		}

		for _, msg := range msgs {
			lastID = msg.ID
			if err := handler(ctx, msg); err != nil {
				log.Warn("redis stream handler error",
					zap.String("stream", streamKey),
					zap.String("msg_id", msg.ID),
					zap.Error(err))
			}
		}
	}
}
