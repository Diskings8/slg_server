package redisstream

import "fmt"

// Stream / PubSub key 前缀
const keyPrefix = "slg:"

// 预定义的 Redis Stream key
const (
	// StreamKeyMarchEvents 行军事件 — worldmap 发布 → game 消费
	StreamKeyMarchEvents = keyPrefix + "march:events"

	// PubSubChannelRoleEnter 角色进服广播频道 — login 发布 → 所有 gateway 订阅（踢旧连接）。
	// 消息体为 pb_redis_stream.RoleEnterEvent 的 proto bytes。
	PubSubChannelRoleEnter = keyPrefix + "gate:role:enter"
)

// RoleGateRouteKey 角色进服路由表 key：value 为 pb_redis_stream.RoleEnterEvent 的 proto bytes。
// login 写（进服）、gateway 可查询（定位 role 所在 gateway，供下推 RPC 用）。
func RoleGateRouteKey(roleID uint64) string {
	return fmt.Sprintf("%sgate:route:%d", keyPrefix, roleID)
}
