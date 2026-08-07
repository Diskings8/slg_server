package login_logics

// 进服业务逻辑：EnterServer（建角 / 进入已有角色）+ 进服后广播路由（踢旧连接）。

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_game_clients"
	"server.slg.com/services/login/login_internals/login_role_routes"
	"server.slg.com/services/login/login_internals/login_servers_store"
	"server.slg.com/services/login/login_internals/login_tokens"
	"server.slg.com/services/login/login_models"
)

// EnterServer 进入区服：role_id=0 新建角色（调 game 建角 + 写映射），非 0 校验归属。
// 成功后记录角色路由表 + 广播（gateway 踢掉该 role 旧连接）。
func EnterServer(ctx context.Context, req *pb_account.EnterServerReq) (*pb_account.EnterServerResp, error) {
	if req == nil || req.GetAccountId() == 0 || req.GetServerId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id and server_id required")
	}
	if err := requireStore(); err != nil {
		return nil, err
	}
	if !login_tokens.Get().Verify(req.GetAccountId(), req.GetToken()) {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	acc, err := login_accounts.Get().GetAccountByID(req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
	}
	if acc == nil {
		return nil, status.Error(codes.NotFound, "account not found")
	}

	sv, err := login_servers_store.Get().GetServer(req.GetServerId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query server failed: %v", err)
	}
	if sv == nil {
		return nil, status.Error(codes.NotFound, "server not found")
	}
	if sv.Status != 0 {
		return nil, status.Error(codes.FailedPrecondition, "server in maintenance")
	}

	var role *login_models.Role
	if req.GetRoleId() == 0 {
		role, err = createNewRole(ctx, req)
	} else {
		role, err = enterExistingRole(acc.ID, req.GetServerId(), req.GetRoleId())
	}
	if err != nil {
		return nil, err
	}

	// 更新最近登录（供角色列表 last_use 填充）
	if err := login_accounts.Get().UpdateLastLogin(acc.ID, sv.ID, role.RoleID); err != nil {
		return nil, status.Errorf(codes.Internal, "update last login failed: %v", err)
	}

	// 进服成功 → 记录角色路由表 + 广播（gateway 踢掉该 role 旧连接）。
	// gateway 节点标识从 ctx 读取（Do 注入 withGatewayNodeID）。
	if err := login_role_routes.PublishRoleEnter(ctx, role.RoleID, sv.ID, GatewayNodeIDFrom(ctx)); err != nil {
		loggers.Logger.Warn("publish role enter failed",
			zap.Uint64("role_id", role.RoleID), zap.Error(err))
	}

	return &pb_account.EnterServerResp{
		AccountId: acc.ID,
		ServerId:  sv.ID,
		RoleId:    role.RoleID,
		RoleInfo:  roleToSimpleInfo(sv, role),
	}, nil
}

// createNewRole 新建角色：分配 roleID → 调 game 建角（失败则无脏写）→ 成功后才写映射
func createNewRole(ctx context.Context, req *pb_account.EnterServerReq) (*login_models.Role, error) {
	if req.GetRoleName() == "" {
		return nil, status.Error(codes.InvalidArgument, "role_name required when creating role")
	}

	// 服内角色名唯一
	existing, err := login_accounts.Get().GetRoleByName(req.GetServerId(), req.GetRoleName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "role name already exists")
	}
	// 每账号每服最多一个角色
	existing, err = login_accounts.Get().GetRoleByAccountServer(req.GetAccountId(), req.GetServerId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "account already has role on this server")
	}

	roleID := snowflakes.GenUUID()

	// 先调 game 建角：失败则直接返回，login 侧未写入任何映射（干净）；gameClient 已由 requireStore 校验
	if _, err := login_game_clients.Get().CreateRole(ctx, req.GetServerId(), &pb_common.CreateRoleReq{
		RoleId:   roleID,
		RoleName: req.GetRoleName(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "game create role failed: %v", err)
	}

	// game 成功后才写映射；撞唯一索引说明并发下重名，返回 AlreadyExists
	role := &login_models.Role{
		RoleID:    roleID,
		AccountID: req.GetAccountId(),
		ServerID:  req.GetServerId(),
		RoleName:  req.GetRoleName(),
	}
	if err := login_accounts.Get().CreateRole(role); err != nil {
		if err == login_accounts.ErrRoleExists {
			return nil, status.Error(codes.AlreadyExists, "role name already exists")
		}
		// 映射写失败（本地 DB 错误）：game 侧留下孤儿角色，记录日志，无害
		loggers.Logger.Warn("create role mapping failed, orphan role in game",
			zap.Uint64("role_id", roleID), zap.Error(err))
		return nil, status.Errorf(codes.Internal, "create role mapping failed: %v", err)
	}
	return role, nil
}

// enterExistingRole 进入已有角色：校验该账号在该服确实拥有该角色
func enterExistingRole(accountID uint64, serverID uint32, roleID uint64) (*login_models.Role, error) {
	role, err := login_accounts.Get().GetRoleByAccountServer(accountID, serverID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if role == nil || role.RoleID != roleID {
		return nil, status.Error(codes.PermissionDenied, "role not owned by account on this server")
	}
	return role, nil
}
