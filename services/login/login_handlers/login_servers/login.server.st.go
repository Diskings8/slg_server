package login_servers

import (
	"server.slg.com/api/protocol/pb/pb_account"
)

// LoginServerHandler 登录 RPC 服务（CreateAccount/LoginAccount/ServerList/EnterServer + Do）
var LoginServerHandler = &LoginServer{}

// LoginServer 账号服务门面：实现 AccountService RPC。
// 无状态：业务在 login_logics（包级函数，依赖全为包级单例），本层只做 gRPC 适配 + Do 路由。
// 内嵌 UnimplementedAccountServiceServer 提供 gRPC 向前兼容（mustEmbed 未导出方法只能由此提供）；
// 4 个业务方法在 login.server.func.go 显式转发（无法内嵌消除：Unimplemented 自带同名方法 → ambiguous）。
type LoginServer struct {
	pb_account.UnimplementedAccountServiceServer
}
