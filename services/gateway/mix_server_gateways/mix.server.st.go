package mix_server_gateways

import (
	"context"
	"fmt"
	"io"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_gateway"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"

	"server.slg.com/common/loggers"
	"server.slg.com/services/gateway/session_gateways"
)

var _ gsi.GRPCServiceI = (*MixServer)(nil)

// MixServer 网关混合服务，实现 GatewayService gRPC 接口，用于与其他节点通信
type MixServer struct {
	pb_gateway.UnimplementedGatewayServiceServer
}

func (m *MixServer) ServiceName() string {
	return "Gateway"
}

func (m *MixServer) Register(srv *grpc.Server) {
	pb_gateway.RegisterGatewayServiceServer(srv, m)
}

// Stream 下推流：其他节点（game/worldmap 等）连接后按 roleID 推送消息给客户端。
//
// 上行：NodePacket{RoleId, MsgId, Message{Body}} → 定位该角色 TCP 会话 → 回写客户端（seq=0 推送）
// 下行：本服务当前不回发（预留 ack / 扩展）
func (m *MixServer) Stream(stream pb_gateway.GatewayService_StreamServer) error {
	for {
		packet, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			loggers.Logger.Warn("gateway push stream recv failed", zap.Error(err))
			return err
		}
		if packet == nil || packet.GetRoleId() < 1 {
			continue
		}
		if err := session_gateways.PushToRoleID(packet.GetRoleId(), packet); err != nil {
			loggers.Logger.Warn("gateway push to role failed",
				zap.Uint64("role_id", packet.GetRoleId()), zap.Error(err))
		}
	}
}

func (m *MixServer) NotifyInfo(ctx context.Context, req *pb_common.NotifyInfoReq) (*pb_common.NotifyInfoRsp, error) {
	loggers.Logger.Info(fmt.Sprintf("[gateway] NotifyInfo: %s", req.GetInfo()))
	return &pb_common.NotifyInfoRsp{Result: true}, nil
}
