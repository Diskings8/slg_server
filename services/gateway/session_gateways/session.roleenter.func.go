package session_gateways

// 角色进服广播监听：login EnterServer 成功后发布，所有 gateway 订阅。
// 本机若持有该 role 的旧连接 → 断开（保证同一角色全服只有一处有效连接）。

import (
	"context"
	"strconv"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
)

// StartRoleEnterWatcher 启动进服广播订阅（监听全局 ctx，服务关停时退出）。
// main.go 在 redis 初始化后调用。
func StartRoleEnterWatcher(ctx context.Context) {
	sub := cacheconn.Get().Subscribe(ctx, redisstream.PubSubChannelRoleEnter)
	go func() {
		defer sub.Close()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			msg, err := sub.ReceiveMessage(ctx)
			if err != nil {
				// ctx 取消则退出；否则连接抖动，重连重试
				select {
				case <-ctx.Done():
					return
				default:
					loggers.Logger.Warn("role enter subscribe failed, retry", zap.Error(err))
					time.Sleep(time.Second)
					continue
				}
			}
			handleRoleEnter(msg.Payload)
		}
	}()
}

// handleRoleEnter 处理进服广播：本机持有该 role 会话则踢掉旧连接。
//
// 统一语义：广播 = 该 role 出现新的权威连接，本机旧的连接一律退位
// （同时覆盖跨 gateway 异设备登录、同 gateway 重连/双开两种场景）。
func handleRoleEnter(payload string) {
	// 消息体为 pb_redis_stream.RoleEnterEvent（login 侧 proto.Marshal 发布）
	var event pb_redis_stream.RoleEnterEvent
	if err := proto.Unmarshal([]byte(payload), &event); err != nil || event.GetRoleId() < 1 {
		return
	}

	s := defaultSessionManager.Get(event.GetRoleId())
	if s == nil {
		return // 本机无该 role 连接，无需处理
	}

	loggers.Logger.Info("kick old role connection",
		zap.Uint64("role_id", event.GetRoleId()),
		zap.String("server_id", strconv.FormatUint(uint64(event.GetServerId()), 10)),
		zap.String("new_gateway", event.GetGatewayNodeId()))

	// 断开旧连接（幂等）：关 game 流 + 断 TCP → read loop 退出 → 注销
	s.cleanupGameStream()
	_ = s.conn.Close()
	defaultSessionManager.Unregister(event.GetRoleId())
}
