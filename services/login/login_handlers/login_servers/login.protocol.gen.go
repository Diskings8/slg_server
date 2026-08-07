package login_servers

// 登录协议注册 — 与 game 的 game_handlers/game.protocol.gen.go 同构（集中 RegisterProto）。
//
// 注册动作放在 handler 所在包（而非 login_handlers）：
// login_servers 已依赖 login_handlers（Do 路由），若 login_handlers 反向引用本包会造成循环依赖。
// 新协议两步：protocol.proto 加 MsgID → 此文件 RegisterProto()。

import (
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/services/login/login_handlers"
)

func init() {
	login_handlers.RegisterProto(pb_protocol.MsgID_LoginCreateAccount, &login_handlers.LoginProtoHandler{
		F:   login_handlers.Wrap(LoginServerHandler.CreateAccount),
		Req: &pb_account.CreateAccountReq{},
	})
	login_handlers.RegisterProto(pb_protocol.MsgID_LoginAccount, &login_handlers.LoginProtoHandler{
		F:   login_handlers.Wrap(LoginServerHandler.LoginAccount),
		Req: &pb_account.LoginAccountReq{},
	})
	login_handlers.RegisterProto(pb_protocol.MsgID_LoginServerList, &login_handlers.LoginProtoHandler{
		F:   login_handlers.Wrap(LoginServerHandler.ServerList),
		Req: &pb_account.ServerListReq{},
	})
	// 进服广播逻辑已在 login_logics.EnterServer 内（成功后发布路由）
	login_handlers.RegisterProto(pb_protocol.MsgID_LoginEnterServer, &login_handlers.LoginProtoHandler{
		F:   login_handlers.Wrap(LoginServerHandler.EnterServer),
		Req: &pb_account.EnterServerReq{},
	})
}
