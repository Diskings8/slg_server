package login_servers

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_models"
)

// passwordSalt 密码哈希盐（固定值，仅降低彩虹表风险；生产环境应换 bcrypt/argon2）
const passwordSalt = "slg_login_salt"

// hashPassword 账号密码哈希：md5(salt + account_name + ":" + password)
func hashPassword(name, pwd string) string {
	sum := md5.Sum([]byte(passwordSalt + name + ":" + pwd))
	return hex.EncodeToString(sum[:])
}

// requireStore 依赖注入校验
func (s *LoginServer) requireStore() error {
	if s.accountStore == nil || s.channelStore == nil || s.serverStore == nil || s.tokens == nil {
		return status.Error(codes.Internal, "stores not initialized")
	}
	return nil
}

// CreateAccount 注册账号：account_name 全局唯一（游戏侧账号身份），并建立首个渠道绑定
func (s *LoginServer) CreateAccount(ctx context.Context, req *pb_account.CreateAccountReq) (*pb_account.CreateAccountResp, error) {
	if req == nil || req.GetAccountName() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "account_name and password required")
	}
	if err := s.requireStore(); err != nil {
		return nil, err
	}

	channelType := int32(req.GetChannelType())
	if ch, err := s.channelStore.GetChannel(channelType); err != nil {
		return nil, status.Errorf(codes.Internal, "query channel failed: %v", err)
	} else if ch == nil {
		return nil, status.Error(codes.NotFound, "channel not declared")
	}

	// account_name 全局唯一
	existing, err := s.accountStore.GetAccountByName(req.GetAccountName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "account_name already exists")
	}

	acc := &login_models.Account{
		AccountName:  req.GetAccountName(),
		PasswordHash: hashPassword(req.GetAccountName(), req.GetPassword()),
	}
	binding := &login_models.ChannelAccount{
		ChannelType:    channelType,
		ChannelAccount: req.GetAccountName(), // 首渠道绑定：渠道账号 = 游戏账号名
		AuthInfo:       "",
	}
	if err := s.accountStore.CreateAccountWithChannel(acc, binding); err != nil {
		if err == login_accounts.ErrChannelExists {
			return nil, status.Error(codes.AlreadyExists, "channel account already bound")
		}
		return nil, status.Errorf(codes.Internal, "create account failed: %v", err)
	}

	return &pb_account.CreateAccountResp{
		AccountId: acc.ID,
		RoleList:  &pb_account.RoleSelectList{},
		Token:     s.tokens.Issue(acc.ID),
	}, nil
}

// LoginAccount 登录账号
//
// ① 渠道绑定 (channelType, account_name) 命中 → 校验账号密码 + 刷新 auth_info；
// ② 未命中 → 按全局 account_name + 密码匹配已有账号 → 命中则自动绑定该渠道（多渠道入口）；
// ③ 都未命中 → Unauthenticated。
func (s *LoginServer) LoginAccount(ctx context.Context, req *pb_account.LoginAccountReq) (*pb_account.LoginAccountResp, error) {
	if req == nil || req.GetAccountName() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "account_name and password required")
	}
	if err := s.requireStore(); err != nil {
		return nil, err
	}

	channelType := int32(req.GetChannelType())
	if ch, err := s.channelStore.GetChannel(channelType); err != nil {
		return nil, status.Errorf(codes.Internal, "query channel failed: %v", err)
	} else if ch == nil {
		return nil, status.Error(codes.NotFound, "channel not declared")
	}

	acc, err := s.resolveAccount(ctx, channelType, req.GetAccountName(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &pb_account.LoginAccountResp{
		AccountId: acc.ID,
		RoleList:  s.buildRoleSelectList(acc),
		Token:     s.tokens.Issue(acc.ID),
	}, nil
}

// resolveAccount 解析登录到账号，包含渠道绑定命中 / 自动绑定两种路径
func (s *LoginServer) resolveAccount(ctx context.Context, channelType int32, name, pwd string) (*login_models.Account, error) {
	binding, err := s.accountStore.GetChannel(channelType, name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query channel binding failed: %v", err)
	}
	if binding != nil {
		acc, err := s.accountStore.GetAccountByID(binding.AccountID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
		}
		if acc == nil || acc.PasswordHash != hashPassword(name, pwd) {
			return nil, status.Error(codes.Unauthenticated, "account or password incorrect")
		}
		// 留痕：记录该渠道本次提供的认证信息
		if binding.AuthInfo != pwd {
			if err := s.accountStore.UpdateChannelAuthInfo(binding.ID, pwd); err != nil {
				loggers.Logger.Warn("update channel auth_info failed",
					zap.Uint64("binding_id", binding.ID), zap.Error(err))
			}
		}
		return acc, nil
	}

	// 未命中绑定：按全局账号名 + 密码匹配，命中则自动绑定该渠道
	acc, err := s.accountStore.GetAccountByName(name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
	}
	if acc == nil || acc.PasswordHash != hashPassword(name, pwd) {
		return nil, status.Error(codes.Unauthenticated, "account or password incorrect")
	}
	if err := s.accountStore.CreateChannel(&login_models.ChannelAccount{
		AccountID:      acc.ID,
		ChannelType:    channelType,
		ChannelAccount: name,
		AuthInfo:       pwd,
	}); err != nil && err != login_accounts.ErrChannelExists {
		return nil, status.Errorf(codes.Internal, "bind channel failed: %v", err)
	}
	return acc, nil
}

// ServerList 区服列表（客户端选服）
func (s *LoginServer) ServerList(ctx context.Context, req *pb_account.ServerListReq) (*pb_account.ServerListResp, error) {
	if err := s.requireStore(); err != nil {
		return nil, err
	}

	servers, err := s.serverStore.ListServers()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list servers failed: %v", err)
	}

	resp := &pb_account.ServerListResp{}
	for _, sv := range servers {
		resp.Servers = append(resp.Servers, &pb_account.ServerSimpleInfo{
			ServerId:   sv.ID,
			ServerName: sv.ServerName,
			Status:     sv.Status,
			OpenTime:   sv.OpenTime,
		})
	}
	return resp, nil
}

// EnterServer 进入区服：role_id=0 新建角色（调 game 建角 + 写映射），非 0 校验归属
func (s *LoginServer) EnterServer(ctx context.Context, req *pb_account.EnterServerReq) (*pb_account.EnterServerResp, error) {
	if req == nil || req.GetAccountId() == 0 || req.GetServerId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "account_id and server_id required")
	}
	if err := s.requireStore(); err != nil {
		return nil, err
	}
	if !s.tokens.Verify(req.GetAccountId(), req.GetToken()) {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	acc, err := s.accountStore.GetAccountByID(req.GetAccountId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
	}
	if acc == nil {
		return nil, status.Error(codes.NotFound, "account not found")
	}

	sv, err := s.serverStore.GetServer(req.GetServerId())
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
		role, err = s.createNewRole(ctx, req)
	} else {
		role, err = s.enterExistingRole(acc.ID, req.GetServerId(), req.GetRoleId())
	}
	if err != nil {
		return nil, err
	}

	// 更新最近登录（供角色列表 last_use 填充）
	if err := s.accountStore.UpdateLastLogin(acc.ID, sv.ID, role.RoleID); err != nil {
		return nil, status.Errorf(codes.Internal, "update last login failed: %v", err)
	}

	return &pb_account.EnterServerResp{
		AccountId: acc.ID,
		ServerId:  sv.ID,
		RoleId:    role.RoleID,
		RoleInfo:  roleToSimpleInfo(sv, role),
	}, nil
}

// createNewRole 新建角色：分配 roleID → 调 game 建角（失败则无脏写）→ 成功后才写映射
func (s *LoginServer) createNewRole(ctx context.Context, req *pb_account.EnterServerReq) (*login_models.Role, error) {
	if req.GetRoleName() == "" {
		return nil, status.Error(codes.InvalidArgument, "role_name required when creating role")
	}

	// 服内角色名唯一
	existing, err := s.accountStore.GetRoleByName(req.GetServerId(), req.GetRoleName())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "role name already exists")
	}
	// 每账号每服最多一个角色
	existing, err = s.accountStore.GetRoleByAccountServer(req.GetAccountId(), req.GetServerId())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if existing != nil {
		return nil, status.Error(codes.AlreadyExists, "account already has role on this server")
	}

	roleID := snowflakes.GenUUID()

	// 先调 game 建角：失败则直接返回，login 侧未写入任何映射（干净）
	if s.gameClient == nil {
		return nil, status.Error(codes.Internal, "game client not initialized")
	}
	if _, err := s.gameClient.CreateRole(ctx, req.GetServerId(), &pb_common.CreateRoleReq{
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
	if err := s.accountStore.CreateRole(role); err != nil {
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
func (s *LoginServer) enterExistingRole(accountID uint64, serverID uint32, roleID uint64) (*login_models.Role, error) {
	role, err := s.accountStore.GetRoleByAccountServer(accountID, serverID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query role failed: %v", err)
	}
	if role == nil || role.RoleID != roleID {
		return nil, status.Error(codes.PermissionDenied, "role not owned by account on this server")
	}
	return role, nil
}

// buildRoleSelectList 构建账号名下角色列表，last_use 取最近登录（last_login_server_id + last_login_role_id）的角色
func (s *LoginServer) buildRoleSelectList(acc *login_models.Account) *pb_account.RoleSelectList {
	roles, err := s.accountStore.GetRolesByAccount(acc.ID)
	if err != nil {
		loggers.Logger.Warn("query roles failed",
			zap.Uint64("account_id", acc.ID), zap.Error(err))
		return &pb_account.RoleSelectList{}
	}

	list := &pb_account.RoleSelectList{}
	for _, r := range roles {
		sv, err := s.serverStore.GetServer(r.ServerID)
		if err != nil || sv == nil {
			loggers.Logger.Warn("role server not found",
				zap.Uint32("server_id", r.ServerID), zap.Error(err))
			continue
		}
		info := roleToSimpleInfo(sv, r)
		list.SimpleInfo = append(list.SimpleInfo, info)
		if r.ServerID == acc.LastLoginServerID && r.RoleID == acc.LastLoginRoleID {
			list.LastUse = info
		}
	}
	return list
}

// roleToSimpleInfo 角色映射 → 登录协议的角色简单信息
func roleToSimpleInfo(sv *login_models.Server, r *login_models.Role) *pb_role.RoleSimpleInfo {
	return &pb_role.RoleSimpleInfo{
		ServerId:   r.ServerID,
		ServerName: sv.ServerName,
		RoleId:     r.RoleID,
		RoleName:   r.RoleName,
	}
}
