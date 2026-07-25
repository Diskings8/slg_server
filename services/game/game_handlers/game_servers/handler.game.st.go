package game_servers

import (
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb/pb_game"
)

type HandlerServer struct {
	pb_game.UnimplementedGameHandlerServer
}

// ServiceName 服务名称
func (m *HandlerServer) ServiceName() string {
	return "Game_HandlerServer"
}

// Register 注册到 gRPC 服务器
func (m *HandlerServer) Register(srv *grpc.Server) {
	pb_game.RegisterGameHandlerServer(srv, m)
}
