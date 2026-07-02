package mix_server_games

import (
	"fmt"

	"server.slg.com/api/protocol/pb"
	"server.slg.com/common/loggers"
)

func (m *GameServer) Stream(stream pb.GameNodeService_StreamServer) error {
	for {
		packet, err := stream.Recv()
		if err != nil {
			return err
		}
		loggers.Logger.Info(fmt.Sprintf("[game] Stream recv: msgId=%d", packet.GetMsgId()))

		if err := stream.Send(&pb.NodePacket{MsgId: packet.GetMsgId(), Body: packet.GetBody()}); err != nil {
			return err
		}
	}
}
