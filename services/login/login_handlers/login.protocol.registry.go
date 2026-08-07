package login_handlers

// 登录协议注册表（protomap）— 与 game 的 game_handlers/game.protocol.registry.go 同构。
// Do（login_servers）按 MsgID 路由到注册的处理器，替代手写 switch-case。
//
// 注册动作在 login_servers 包 init()（login.protocol.gen.go）完成——handler 与注册表分属
// 父子包，若本包引用 handler 会造成循环依赖。

import (
	"context"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_protocol"
)

// LoginProtoHandleFunc 登录协议处理函数：反序列化请求 → 执行业务 → 返回响应 + gRPC 错误
type LoginProtoHandleFunc func(ctx context.Context, req proto.Message) (proto.Message, error)

// LoginProtoHandler 协议处理器注册项
type LoginProtoHandler struct {
	F   LoginProtoHandleFunc
	Req proto.Message
}

// loginProtoRegistry 协议号 → 处理器映射表
var loginProtoRegistry = map[pb_protocol.MsgID]*LoginProtoHandler{}

// RegisterProto 注册登录协议处理器
func RegisterProto(msgID pb_protocol.MsgID, handler *LoginProtoHandler) {
	loginProtoRegistry[msgID] = handler
}

// GetProtoHandler 获取登录协议处理器
func GetProtoHandler(msgID pb_protocol.MsgID) (*LoginProtoHandler, bool) {
	h, ok := loginProtoRegistry[msgID]
	return h, ok
}

// Wrap 将类型安全的处理函数包装为 LoginProtoHandleFunc，省去手动 type assertion
func Wrap[Req, Resp proto.Message](fn func(ctx context.Context, req Req) (Resp, error)) LoginProtoHandleFunc {
	return func(ctx context.Context, req proto.Message) (proto.Message, error) {
		return fn(ctx, req.(Req))
	}
}
