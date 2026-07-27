package game_handlers

import (
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/conns/rpcconn/rpc_results"
)

var GameStreamHandler = &GameStream{}

type GameStream struct {
	pb_game.UnimplementedGameServiceServer
}

// Stream GameService 流式连接入口
func (gs *GameStream) Stream(stream pb_game.GameService_StreamServer) error {
	req, err := stream.Recv()
	if err != nil {
		return err
	}
	if req.GetRoleId() < 1 {
		return rpc_results.ErrorParam("参数错误")
	}
	return gs.gateConnectDo(req.GetRoleId(), stream)
}

func (gs *GameStream) gateConnectDo(id uint64, stream pb_game.GameService_StreamServer) error {
	return nil
}
