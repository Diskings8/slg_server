package login_logics

// 请求级上下文：gateway 节点标识（login_servers.Do 注入，进服广播等需要感知来源节点的逻辑读取）。
// 对标 game 把请求元数据放 ctx 而非 Do 层特判。

import "context"

// gatewayNodeIDKey ctx 内 gateway 节点标识的 key
type gatewayNodeIDKey struct{}

// WithGatewayNodeID 向 ctx 注入 gateway 节点标识（login_servers.Do 调用）
func WithGatewayNodeID(ctx context.Context, gatewayNodeID string) context.Context {
	return context.WithValue(ctx, gatewayNodeIDKey{}, gatewayNodeID)
}

// GatewayNodeIDFrom 从 ctx 读取 gateway 节点标识（未注入返回空串）
func GatewayNodeIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(gatewayNodeIDKey{}).(string); ok {
		return v
	}
	return ""
}
