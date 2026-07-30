package game_handlers

import (
	"context"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/rpcconn/rpc_results"
)

// ProtoHandleFunc 协议处理函数
//   - ctx: 上下文
//   - roleID: 角色ID
//   - req: 反序列化后的请求 proto
//   - resp: 预创建的响应 proto（worldmap_handlers 填充字段后返回）
//
// 返回 nil 表示成功，返回 ResultI 表示业务错误
type ProtoHandleFunc func(ctx context.Context, roleID uint64, req proto.Message, resp proto.Message) rpc_results.ResultI

// ProtoHandler 协议处理器注册项
type ProtoHandler struct {
	F    ProtoHandleFunc
	Req  proto.Message
	Resp proto.Message
}

// protoRegistry 协议号 → 处理器映射表
var protoRegistry = map[pb_protocol.MsgID]*ProtoHandler{}

// RegisterProto 注册协议处理器
func RegisterProto(msgID pb_protocol.MsgID, handler *ProtoHandler) {
	protoRegistry[msgID] = handler
}

// GetProtoHandler 获取协议处理器
func GetProtoHandler(msgID pb_protocol.MsgID) (*ProtoHandler, bool) {
	h, ok := protoRegistry[msgID]
	return h, ok
}

// Wrap 将类型安全的处理函数包装为 ProtoHandleFunc，省去手动 type assertion
//
//	worldmap_handlers := Wrap(hero_handler.HandlerHeroList)
//	// 等价于:
//	// func(ctx, roleID, req proto.Message, resp proto.Message) rpc_results.ResultI {
//	//     return hero_handler.HandlerHeroList(ctx, roleID, req.(*pb_hero.HeroListReq), resp.(*pb_hero.HeroListResp))
//	// }
func Wrap[Req, Resp proto.Message](fn func(ctx context.Context, roleID uint64, req Req, resp Resp) rpc_results.ResultI) ProtoHandleFunc {
	return func(ctx context.Context, roleID uint64, req proto.Message, resp proto.Message) rpc_results.ResultI {
		return fn(ctx, roleID, req.(Req), resp.(Resp))
	}
}
