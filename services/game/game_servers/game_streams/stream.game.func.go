package game_streams

import (
	"errors"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_connects"
)

func (s *StreamServer) Stream(stream pb_game.GameService_StreamServer) error {
	packet, err := stream.Recv() // 接收认证信息
	if err != nil {
		return err
	}
	if packet.GetMsgId() != pb_protocol.MsgID_Common_Auth {
		return errors.New("invalid Auth MsgID")
	}
	tf := &pb_role.RoleTransfer{}
	if errUnmarshal := proto.Unmarshal(packet.GetBody(), tf); errUnmarshal != nil {
		return errUnmarshal
	}
	// todo
	_ = map_connects.NewRoleConnect(tf.GetRoleId(), cores_declarations.MapID(tf.GetCityMapId()), stream)

	return nil
}
