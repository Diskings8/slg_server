package game_handlers

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/loggers"
)

func (s *HandlerServer) CreateRole(ctx context.Context, req *pb_common.CreateRoleReq) (*pb_common.CreateRoleRsp, error) {
	loggers.Logger.Info(fmt.Sprintf("[game] CreateRole: userId=%d, roleName=%s", req.GetUserId(), req.GetRoleName()))
	return &pb_common.CreateRoleRsp{RoleId: req.GetUserId()}, nil
}

func (s *HandlerServer) LoginOnce(ctx context.Context, req *pb_common.LoginOnceReq) (*pb_common.LoginOnceRsp, error) {
	loggers.Logger.Info(fmt.Sprintf("[game] LoginOnce: roleId=%d", req.GetRoleId()))
	return &pb_common.LoginOnceRsp{Result: true}, nil
}
