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
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/loggers"
)

// Do 统一协议入口：按 MsgID 反序列化并调用对应 RPC，错误映射为登录错误码
func (s *LoginServer) Do(ctx context.Context, req *pb_common.NodePacket) (*pb_common.NodePacket, error) {
	msgID := req.GetMsgId()

	var (
		reqPB proto.Message
		call  func(context.Context, proto.Message) (proto.Message, error)
	)
	switch msgID {
	case pb_protocol.MsgID_LoginCreateAccount:
		reqPB = &pb_account.CreateAccountReq{}
		call = func(ctx context.Context, r proto.Message) (proto.Message, error) {
			return s.CreateAccount(ctx, r.(*pb_account.CreateAccountReq))
		}
	case pb_protocol.MsgID_LoginAccount:
		reqPB = &pb_account.LoginAccountReq{}
		call = func(ctx context.Context, r proto.Message) (proto.Message, error) {
			return s.LoginAccount(ctx, r.(*pb_account.LoginAccountReq))
		}
	case pb_protocol.MsgID_LoginServerList:
		reqPB = &pb_account.ServerListReq{}
		call = func(ctx context.Context, r proto.Message) (proto.Message, error) {
			return s.ServerList(ctx, r.(*pb_account.ServerListReq))
		}
	case pb_protocol.MsgID_LoginEnterServer:
		reqPB = &pb_account.EnterServerReq{}
		call = func(ctx context.Context, r proto.Message) (proto.Message, error) {
			return s.EnterServer(ctx, r.(*pb_account.EnterServerReq))
		}
	default:
		loggers.Logger.Warn("login protocol not found", zap.Int32("msg_id", int32(msgID)))
		return &pb_common.NodePacket{
			MsgId: msgID,
			Message: &pb_common.MessagePacket{
				ErrCode: pb_error_code.ErrorCode_ProtocolNotFound,
			},
		}, nil
	}

	// 反序列化请求体
	if err := proto.Unmarshal(req.GetMessage().GetBody(), reqPB); err != nil {
		return &pb_common.NodePacket{
			MsgId: msgID,
			Message: &pb_common.MessagePacket{
				ErrCode: pb_error_code.ErrorCode_ParamError,
				DevMsg:  err.Error(),
			},
		}, nil
	}

	// 执行业务
	resp, err := call(ctx, reqPB)
	if err != nil {
		return errorNodePacket(msgID, err), nil
	}

	respBytes, err := proto.Marshal(resp)
	if err != nil {
		return &pb_common.NodePacket{
			MsgId: msgID,
			Message: &pb_common.MessagePacket{
				ErrCode: pb_error_code.ErrorCode_Failed,
				DevMsg:  err.Error(),
			},
		}, nil
	}

	return &pb_common.NodePacket{
		MsgId: msgID,
		Message: &pb_common.MessagePacket{
			Body: respBytes,
		},
	}, nil
}

// errorNodePacket 构建业务错误响应包
func errorNodePacket(msgID pb_protocol.MsgID, err error) *pb_common.NodePacket {
	return &pb_common.NodePacket{
		MsgId: msgID,
		Message: &pb_common.MessagePacket{
			ErrCode: loginErrorCode(msgID, err),
			DevMsg:  err.Error(),
		},
	}
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
