package mix_server_games

import (
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb"
	gsi "server.slg.com/common/servers/grpc_server_interfaces"
)

var _ gsi.GRPCServiceI = (*GameServer)(nil)

type GameServer struct {
	pb.UnimplementedGameNodeServiceServer
}

func (m *GameServer) ServiceName() string {
	return "Game"
}

func (m *GameServer) Register(srv *grpc.Server) {
	pb.RegisterGameNodeServiceServer(srv, m)
}
