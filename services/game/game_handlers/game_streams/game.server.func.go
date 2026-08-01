package game_streams

import (
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_entitys/game_role_handler"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_handlers"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_internals/gate_stream"
)

// Recv 接收 Gateway 转发的客户端请求并路由到协议处理器
func (gs *GameStream) Recv(stream grpc.ServerStream) error {
	req := &pb_game.GameServiceNodePacketReq{}
	if err := stream.RecvMsg(req); err != nil {
		return err
	}

	packet := req.GetPacket()
	if packet == nil {
		return nil
	}

	msgID := packet.GetMsgId()
	roleID := req.GetRoleId()

	// 相机等地图实时消息 → 转发到玩家的 worldmap 视野流，由 worldmap 处理并下推
	switch msgID {
	case pb_protocol.MsgID_GameCameraInit, pb_protocol.MsgID_GameCameraMove:
		if err := game_rpc_clients.WorldMap().SendToWorldMap(roleID, packet); err != nil {
			return gate_stream.GateCallBackFail(roleID, msgID, pb_error_code.ErrorCode_Failed, err.Error())
		}
		return nil
	}

	handler, ok := game_handlers.GetProtoHandler(msgID)
	if !ok {
		return gate_stream.GateCallBackFail(roleID, msgID, pb_error_code.ErrorCode_ProtocolNotFound, "protocol not found")
	}

	var reqPB proto.Message
	if handler.Req != nil {
		reqPB = proto.Clone(handler.Req)
		if err := proto.Unmarshal(packet.GetMessage().GetBody(), reqPB); err != nil {
			return gate_stream.GateCallBackFail(roleID, msgID, pb_error_code.ErrorCode_ParamError, err.Error())
		}
	}

	rolePoller, _, err := game_role_handler.GetRole(roleID)
	if err != nil {
		return gate_stream.GateCallBackFail(roleID, msgID, err.Code(), err.DevMsg())
	}
	defer rolePoller.Release()

	resp := proto.Clone(handler.Resp)
	if result := handler.F(stream.Context(), roleID, reqPB, resp); result != nil {
		return gate_stream.GateCallBackFail(roleID, msgID, result.Code(), result.DevMsg())
	}

	return gate_stream.GateCallBackSuccess(roleID, msgID, resp)
}

// Offline 角色下线处理
func Offline(role *game_roles.Role) {
	// TODO: 接入其他子模块后补充下线逻辑
	loggers.Logger.Info("role offline", zap.Uint64("role_id", role.ID))
}
