package mix_server_games

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb"
	"server.slg.com/common/loggers"
)

func (m *GameServer) CreateRole(ctx context.Context, req *pb.CreateRoleReq) (*pb.CreateRoleRsp, error) {
	loggers.Log.Info(fmt.Sprintf("[game] CreateRole: userId=%d, roleName=%s", req.GetUserId(), req.GetRoleName()))
	return &pb.CreateRoleRsp{RoleId: req.GetUserId()}, nil
}

func (m *GameServer) LoginOnce(ctx context.Context, req *pb.LoginOnceReq) (*pb.LoginOnceRsp, error) {
	loggers.Log.Info(fmt.Sprintf("[game] LoginOnce: roleId=%d", req.GetRoleId()))
	return &pb.LoginOnceRsp{Result: true}, nil
}
