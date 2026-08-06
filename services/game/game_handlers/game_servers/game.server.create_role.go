package game_servers

import (
	"context"
	"strconv"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/conns/dbconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
	"server.slg.com/services/game/game_logics"
)

// serverID 取本服区服 ID（instance 即区服号，一个 game 进程对应一个区服）
func serverID() uint32 {
	id, _ := strconv.ParseUint(*vgc.CommonGlobalVarInstance, 10, 32)
	return uint32(id)
}

// CreateRole 创建角色（login/account 节点经 gRPC 调用）
//
// roleID 由 login/account 节点分配，game 直接使用。
// 流程：建 Role 实体 → 设 server_id/role_name → 调 worldmap 分配出生点并落主城
// → 建主城建筑 → DBCreate 持久化 → 返回 roleId。
func (s *GameServer) CreateRole(ctx context.Context, req *pb_common.CreateRoleReq) (*pb_common.CreateRoleRsp, error) {
	loggers.Logger.Info("CreateRole",
		zap.Uint64("role_id", req.GetRoleId()),
		zap.String("role_name", req.GetRoleName()))

	// 1. roleID 由 login/account 节点分配
	roleID := req.GetRoleId()
	if roleID == 0 {
		return nil, status.Error(codes.InvalidArgument, "invalid role_id")
	}

	// 2. 构造角色实体（空子模块容器）
	role := &game_roles.Role{ID: roleID}
	role.New()

	// 3. 设置区服ID与角色名，供登录/出征链路读取
	sid := serverID()
	attr := role.GetAttr().Ensure()
	attr.ServerID = sid
	attr.RoleName = req.GetRoleName()

	// 4. 调 worldmap 分配出生点并落主城，拿主城核心坐标
	createRsp, callErr := game_rpc_clients.WorldMap().CreateRole(ctx, &pb_worldmap.CreateRoleReq{
		RoleId:   roleID,
		ServerId: sid,
		RoleName: req.GetRoleName(),
	})
	if callErr != nil {
		return nil, status.Errorf(codes.Internal, "worldmap create role failed: %v", callErr)
	}
	coreMapID := createRsp.GetMapId()

	// 5. 建主城建筑（复用统一入口：占地/ID/校场队列初始化）
	if _, result := game_logics.BuildingBuild(role, roleID, &pb_city.BuildingBuildReq{
		Type:  pb_city.BuildingType_RoleMainCity,
		MapId: coreMapID,
	}); result != nil {
		return nil, status.Errorf(codes.Internal, "build main city failed: %s", result.DevMsg())
	}

	// 6. 持久化（事务：各子模块 DBCreate）
	writeDB := dbconn.GetWriteDbConn()
	if writeDB == nil {
		return nil, status.Error(codes.Internal, "write db not initialized")
	}
	if err := role.DBCreate(writeDB); err != nil {
		return nil, status.Errorf(codes.Internal, "role db create failed: %v", err)
	}

	return &pb_common.CreateRoleRsp{RoleId: roleID}, nil
}
