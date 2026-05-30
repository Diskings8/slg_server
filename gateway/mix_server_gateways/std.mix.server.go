package mix_server_gateways

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"

	"server.slg.com/common/loggers"
)

var _ gsi.GRPCServiceI = (*MixServer)(nil)

type MixServer struct {
	pb.UnimplementedGatewayNodeServiceServer
}

func (m *MixServer) ServiceName() string {
	return "Gateway"
}

func (m *MixServer) Register(srv *grpc.Server) {
	pb.RegisterGatewayNodeServiceServer(srv, m)
}

func (m *MixServer) NotifyInfo(ctx context.Context, req *pb.NotifyInfoReq) (*pb.NotifyInfoRsp, error) {
	loggers.Log.Info(fmt.Sprintf("[gateway] NotifyInfo: %s", req.GetInfo()))
	return &pb.NotifyInfoRsp{Result: true}, nil
}
