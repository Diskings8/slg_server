package login_role_routes

// 角色进服路由：EnterServer 成功后写 Redis 路由表 + 发布广播（所有 gateway 订阅后踢掉旧连接）。

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_redis_stream"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/redisstream"
)

// routeTTL 路由表 TTL：进服会话期间有效，超时兜底清理僵尸条目
const routeTTL = 6 * time.Hour

// PublishRoleEnter 记录角色进服路由 + 广播，通知所有 gateway 踢掉该 role 的旧连接。
//
// login 掌握 {RoleID, ServerID, GatewayNodeID}——EnterServer 成功 + 本请求来自哪个 gateway。
// 消息体用 pb_redis_stream.RoleEnterEvent（proto），与 redis_stream.proto 其他事件一致。
func PublishRoleEnter(ctx context.Context, roleID uint64, serverID uint32, gatewayNodeID string) error {
	if roleID < 1 {
		return nil
	}

	event := &pb_redis_stream.RoleEnterEvent{
		RoleId:        roleID,
		ServerId:      serverID,
		GatewayNodeId: gatewayNodeID,
	}
	data, err := proto.Marshal(event)
	if err != nil {
		return err
	}

	cache := cacheconn.Get()

	// 路由表：roleId → RoleEnterEvent（保留可查询，未来下推 RPC 按 roleId 定位所在 gateway）
	if err := cache.Set(ctx, redisstream.RoleGateRouteKey(roleID), string(data), routeTTL).Err(); err != nil {
		loggers.Logger.Warn("role route set failed", zap.Uint64("role_id", roleID), zap.Error(err))
	}

	// 广播：所有 gateway 订阅后，本机持有该 role 旧连接则踢掉
	if err := cache.Publish(ctx, redisstream.PubSubChannelRoleEnter, string(data)).Err(); err != nil {
		loggers.Logger.Warn("role enter publish failed", zap.Uint64("role_id", roleID), zap.Error(err))
		return err
	}

	return nil
}
