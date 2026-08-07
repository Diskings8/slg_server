package login_servers

// Do — 统一协议入口（gateway 转发入口，与 game 的 GameHandler.Do 同构）。
//
// 请求: NodePacket.msgId = 登录协议号（LoginCreateAccount/LoginAccount/LoginServerList/LoginEnterServer）
// 响应: NodePacket.msgId 回显，message.body = 序列化响应；错误时 err_code = 登录错误码 + dev_msg 透传。

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_nodepacket"
	"server.slg.com/services/login/login_handlers"
	"server.slg.com/services/login/login_logics"
)

// Do 统一协议入口：按 MsgID 经协议注册表（protomap）路由，与 game 的 GameHandler.Do 同构。
// 错误映射为登录错误码（loginErrorCode）。协议无关，不含任何按 MsgID 的特判。
func (s *LoginServer) Do(ctx context.Context, req *pb_common.NodePacket) (*pb_common.NodePacket, error) {
	msgID := req.GetMsgId()

	handler, ok := login_handlers.GetProtoHandler(msgID)
	if !ok {
		loggers.Logger.Warn("login protocol not found", zap.Int32("msg_id", int32(msgID)))
		return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_ProtocolNotFound, ""), nil
	}

	// 反序列化请求体（克隆注册表预创建的空请求实例）
	reqPB := proto.Clone(handler.Req)
	if err := proto.Unmarshal(req.GetMessage().GetBody(), reqPB); err != nil {
		return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_ParamError, err.Error()), nil
	}

	// 请求级上下文：注入 gateway 节点标识（gateway 转发时填 MessagePacket.GatewayNodeId），
	// 供需要感知来源节点的协议逻辑（如进服广播）从 ctx 读取，Do 本身不关心具体协议。
	ctx = login_logics.WithGatewayNodeID(ctx, req.GetMessage().GetGatewayNodeId())

	// 执行业务
	resp, err := handler.F(ctx, reqPB)
	if err != nil {
		return util_nodepacket.Error(msgID, loginErrorCode(msgID, err), err.Error()), nil
	}

	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_Failed, err.Error()), nil
	}

	return util_nodepacket.Success(msgID, respBytes), nil
}

// loginErrorCode 按 MsgID 把 gRPC 状态码映射为登录错误码，供客户端区分失败原因。
//
// 同一 gRPC 码在不同协议下含义不同（如 AlreadyExists 在注册=账号名占用、在进服=角色名占用），
// 因此按 MsgID 上下文映射；dev_msg 携带原始信息兜底。
func loginErrorCode(msgID pb_protocol.MsgID, err error) pb_error_code.ErrorCode {
	st, ok := status.FromError(err)
	if !ok {
		return pb_error_code.ErrorCode_Failed
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return pb_error_code.ErrorCode_ParamError
	case codes.Unauthenticated:
		if msgID == pb_protocol.MsgID_LoginAccount {
			return pb_error_code.ErrorCode_AccountOrPasswordWrong
		}
		return pb_error_code.ErrorCode_TokenInvalid
	case codes.NotFound:
		if msgID == pb_protocol.MsgID_LoginCreateAccount || msgID == pb_protocol.MsgID_LoginAccount {
			return pb_error_code.ErrorCode_ChannelNotDeclared
		}
		return pb_error_code.ErrorCode_ServerNotFound
	case codes.AlreadyExists:
		if msgID == pb_protocol.MsgID_LoginCreateAccount {
			return pb_error_code.ErrorCode_AccountExists
		}
		return pb_error_code.ErrorCode_RoleNameExists
	case codes.PermissionDenied:
		return pb_error_code.ErrorCode_RoleNotOwned
	case codes.FailedPrecondition:
		return pb_error_code.ErrorCode_ServerMaintenance
	default:
		return pb_error_code.ErrorCode_Failed
	}
}
