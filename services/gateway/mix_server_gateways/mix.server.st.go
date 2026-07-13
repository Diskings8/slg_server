package mix_server_gateways

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_gateway"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"

	"server.slg.com/common/loggers"
)

var _ gsi.GRPCServiceI = (*MixServer)(nil)

// MixServer 网关混合服务，实现 GatewayNodeService gRPC 接口，用于与其他节点通信
type MixServer struct {
	pb_gateway.UnimplementedGatewayServiceServer
}

func (m *MixServer) ServiceName() string {
	return "Gateway"
}

func (m *MixServer) Register(srv *grpc.Server) {
	pb_gateway.RegisterGatewayServiceServer(srv, m)
}

func (m *MixServer) NotifyInfo(ctx context.Context, req *pb_common.NotifyInfoReq) (*pb_common.NotifyInfoRsp, error) {
	loggers.Logger.Info(fmt.Sprintf("[gateway] NotifyInfo: %s", req.GetInfo()))
	return &pb_common.NotifyInfoRsp{Result: true}, nil
}
