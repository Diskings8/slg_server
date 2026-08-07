package login_servers

// 4 个 RPC 方法：AccountServiceServer gRPC 接口的强制实现，业务在 login_logics.Logic。
//
// 不能靠内嵌 *login_logics.Logic 消除：AccountServiceServer 要求 mustEmbedUnimplementedAccountServiceServer()
// （未导出方法，只能由嵌入 pb_account.UnimplementedAccountServiceServer 提供），而 Unimplemented 自带
// 同名方法，与 Logic 提升的方法冲突（ambiguous selector）→ 必须显式实现并转发。
// 参数校验已在 login_logics 各方法内统一处理，此处仅做 gRPC 协议适配。

import (
	"context"

	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/services/login/login_logics"
)

// CreateAccount 注册账号
func (s *LoginServer) CreateAccount(ctx context.Context, req *pb_account.CreateAccountReq) (*pb_account.CreateAccountResp, error) {
	return login_logics.CreateAccount(ctx, req)
}

// LoginAccount 登录账号
func (s *LoginServer) LoginAccount(ctx context.Context, req *pb_account.LoginAccountReq) (*pb_account.LoginAccountResp, error) {
	return login_logics.LoginAccount(ctx, req)
}

// ServerList 区服列表
func (s *LoginServer) ServerList(ctx context.Context, req *pb_account.ServerListReq) (*pb_account.ServerListResp, error) {
	return login_logics.ServerList(ctx, req)
}

// EnterServer 进入区服（含进服后广播路由）
func (s *LoginServer) EnterServer(ctx context.Context, req *pb_account.EnterServerReq) (*pb_account.EnterServerResp, error) {
	return login_logics.EnterServer(ctx, req)
}
