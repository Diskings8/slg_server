package game_handlers

import (
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb/pb_game"
)

type HandlerServer struct {
	pb_game.UnimplementedHandlerServer
}

func (m *HandlerServer) ServiceName() string {
	return "Game_HandlerServer"
}

func (m *HandlerServer) Register(srv *grpc.Server) {
	pb_game.RegisterHandlerServer(srv, m)
}
