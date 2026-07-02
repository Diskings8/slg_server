package mix_server_games

import (
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"
)

var _ gsi.GRPCServiceI = (*GameServer)(nil)

// GameServer 游戏混合服务，实现 GameNodeService gRPC 接口，用于接收网关转发的请求
type GameServer struct {
	pb.UnimplementedGameNodeServiceServer
}

func (m *GameServer) ServiceName() string {
	return "Game"
}

func (m *GameServer) Register(srv *grpc.Server) {
	pb.RegisterGameNodeServiceServer(srv, m)
}
