package game_streams

import (
	"google.golang.org/grpc"
	"server.slg.com/api/protocol/pb/pb_game"
)

type StreamServer struct {
	pb_game.UnimplementedGameServiceServer
}

func (s *StreamServer) ServiceName() string {
	return "Game_StreamServer"
}

func (s *StreamServer) Register(srv *grpc.Server) {
	pb_game.RegisterGameServiceServer(srv, s)
}
