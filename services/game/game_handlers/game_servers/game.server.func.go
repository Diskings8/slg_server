package game_servers

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_nodepacket"
	"server.slg.com/services/game/game_handlers"
)

// Do 统一协议入口
//
// 请求:
//
//	NodePacket.msgId = 协议号 (如 1000001)
//	NodePacket.message.body = 序列化的请求 proto
//
// 响应:
//
//	NodePacket.msgId = 协议号
//	NodePacket.message.body = 序列化的响应 proto (成功时)
//	NodePacket.message.err_code = 0=成功, >0=错误码
func (s *GameServer) Do(ctx context.Context, req *pb_common.NodePacket) (*pb_common.NodePacket, error) {
	msgID := req.GetMsgId()

	handler, ok := game_handlers.GetProtoHandler(msgID)
	if !ok {
		loggers.Logger.Warn(fmt.Sprintf("protocol not found, msgId: %d", msgID))
		return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_ProtocolNotFound, ""), nil
	}

	// 反序列化请求
	var reqPB proto.Message
	if handler.Req != nil {
		reqPB = proto.Clone(handler.Req)
		if err := proto.Unmarshal(req.GetMessage().GetBody(), reqPB); err != nil {
			loggers.Logger.Error(fmt.Sprintf("proto.Unmarshal failed, msgId: %d, error: %s", msgID, err.Error()))
			return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_ParamError, err.Error()), nil
		}
	}

	// 从请求中获取角色ID
	roleID := req.GetRoleId()

	// 执行业务
	resp := proto.Clone(handler.Resp)
	if result := handler.F(ctx, roleID, reqPB, resp); result != nil {
		return util_nodepacket.Error(msgID, result.Code(), result.DevMsg()), nil
	}

	// 成功
	respBytes, err := proto.Marshal(resp)
	if err != nil {
		loggers.Logger.Error(fmt.Sprintf("proto.Marshal failed, msgId: %d, error: %s", msgID, err.Error()))
		return util_nodepacket.Error(msgID, pb_error_code.ErrorCode_Failed, ""), nil
	}

	return util_nodepacket.Success(msgID, respBytes), nil
}
