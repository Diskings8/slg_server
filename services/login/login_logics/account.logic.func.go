package login_logics

// 账号业务逻辑：注册 / 登录 / 渠道解析 / 角色列表。

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_role"
	"server.slg.com/common/loggers"
	"server.slg.com/services/login/login_internals/login_accounts"
	"server.slg.com/services/login/login_internals/login_channels"
	"server.slg.com/services/login/login_internals/login_game_clients"
	"server.slg.com/services/login/login_internals/login_servers_store"
	"server.slg.com/services/login/login_internals/login_tokens"
	"server.slg.com/services/login/login_models"
)

// passwordSalt 密码哈希盐（固定值，仅降低彩虹表风险；生产环境应换 bcrypt/argon2）
const passwordSalt = "slg_login_salt"

// hashPassword 账号密码哈希：md5(salt + account_name + ":" + password)
func hashPassword(name, pwd string) string {
	sum := md5.Sum([]byte(passwordSalt + name + ":" + pwd))
	return hex.EncodeToString(sum[:])
}

// requireStore 依赖就绪校验（store/tokens 为包级单例，仅校验注入的 gameClient）
func requireStore() error {
	if login_game_clients.Get() == nil {
		return status.Error(codes.Internal, "game client not initialized")
	}
	return nil
}

// CreateAccount 注册账号：account_name 全局唯一（游戏侧账号身份），并建立首个渠道绑定
func CreateAccount(ctx context.Context, req *pb_account.CreateAccountReq) (*pb_account.CreateAccountResp, error) {
	if req == nil || req.GetAccountName() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "account_name and password required")
	}
	if err := requireStore(); err != nil {
		return nil, err
	}

	channelType := int32(req.GetChannelType())
	if ch, err := login_channels.Get().GetChannel(channelType); err != nil {
		return nil, status.Errorf(codes.Internal, "query channel failed: %v", err)
	} else if ch == nil {
		return nil, status.Error(codes.NotFound, "channel not declared")
	}

	// account_name 全局唯一
	existing, err := login_accounts.Get().GetAccountByName(req.GetAccountName())
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
	if err := login_accounts.Get().CreateAccountWithChannel(acc, binding); err != nil {
		if err == login_accounts.ErrChannelExists {
			return nil, status.Error(codes.AlreadyExists, "channel account already bound")
		}
		return nil, status.Errorf(codes.Internal, "create account failed: %v", err)
	}

	return &pb_account.CreateAccountResp{
		AccountId: acc.ID,
		RoleList:  &pb_account.RoleSelectList{},
		Token:     login_tokens.Get().Issue(acc.ID),
	}, nil
}

// LoginAccount 登录账号
//
// ① 渠道绑定 (channelType, account_name) 命中 → 校验账号密码 + 刷新 auth_info；
// ② 未命中 → 按全局 account_name + 密码匹配已有账号 → 命中则自动绑定该渠道（多渠道入口）；
// ③ 都未命中 → Unauthenticated。
func LoginAccount(ctx context.Context, req *pb_account.LoginAccountReq) (*pb_account.LoginAccountResp, error) {
	if req == nil || req.GetAccountName() == "" || req.GetPassword() == "" {
		return nil, status.Error(codes.InvalidArgument, "account_name and password required")
	}
	if err := requireStore(); err != nil {
		return nil, err
	}

	channelType := int32(req.GetChannelType())
	if ch, err := login_channels.Get().GetChannel(channelType); err != nil {
		return nil, status.Errorf(codes.Internal, "query channel failed: %v", err)
	} else if ch == nil {
		return nil, status.Error(codes.NotFound, "channel not declared")
	}

	acc, err := resolveAccount(ctx, channelType, req.GetAccountName(), req.GetPassword())
	if err != nil {
		return nil, err
	}

	return &pb_account.LoginAccountResp{
		AccountId: acc.ID,
		RoleList:  buildRoleSelectList(acc),
		Token:     login_tokens.Get().Issue(acc.ID),
	}, nil
}

// resolveAccount 解析登录到账号，包含渠道绑定命中 / 自动绑定两种路径
func resolveAccount(ctx context.Context, channelType int32, name, pwd string) (*login_models.Account, error) {
	binding, err := login_accounts.Get().GetChannel(channelType, name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query channel binding failed: %v", err)
	}
	if binding != nil {
		acc, err := login_accounts.Get().GetAccountByID(binding.AccountID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
		}
		if acc == nil || acc.PasswordHash != hashPassword(name, pwd) {
			return nil, status.Error(codes.Unauthenticated, "account or password incorrect")
		}
		// 留痕：记录该渠道本次提供的认证信息
		if binding.AuthInfo != pwd {
			if err := login_accounts.Get().UpdateChannelAuthInfo(binding.ID, pwd); err != nil {
				loggers.Logger.Warn("update channel auth_info failed",
					zap.Uint64("binding_id", binding.ID), zap.Error(err))
			}
		}
		return acc, nil
	}

	// 未命中绑定：按全局账号名 + 密码匹配，命中则自动绑定该渠道
	acc, err := login_accounts.Get().GetAccountByName(name)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query account failed: %v", err)
	}
	if acc == nil || acc.PasswordHash != hashPassword(name, pwd) {
		return nil, status.Error(codes.Unauthenticated, "account or password incorrect")
	}
	if err := login_accounts.Get().CreateChannel(&login_models.ChannelAccount{
		AccountID:      acc.ID,
		ChannelType:    channelType,
		ChannelAccount: name,
		AuthInfo:       pwd,
	}); err != nil && err != login_accounts.ErrChannelExists {
		return nil, status.Errorf(codes.Internal, "bind channel failed: %v", err)
	}
	return acc, nil
}

// buildRoleSelectList 构建账号名下角色列表，last_use 取最近登录（last_login_server_id + last_login_role_id）的角色
func buildRoleSelectList(acc *login_models.Account) *pb_account.RoleSelectList {
	roles, err := login_accounts.Get().GetRolesByAccount(acc.ID)
	if err != nil {
		loggers.Logger.Warn("query roles failed",
			zap.Uint64("account_id", acc.ID), zap.Error(err))
		return &pb_account.RoleSelectList{}
	}

	list := &pb_account.RoleSelectList{}
	for _, r := range roles {
		sv, err := login_servers_store.Get().GetServer(r.ServerID)
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
