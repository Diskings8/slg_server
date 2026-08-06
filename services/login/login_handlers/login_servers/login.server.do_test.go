//go:build integration

package login_servers_test

// login.Do 统一协议入口单测 — 连真实 mysql（common_db_0）+ mock game。
// 运行：go test -tags integration ./services/login/...
// 覆盖：4 个协议的路由 / 成功序列化 / 错误码映射（重复注册、错密码、token 无效、未知协议、非法 body）。

import (
	"context"
	"testing"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/services/login/login_testutil"
)

func TestLoginDo(t *testing.T) {
	game := &fakeGameClient{}
	cli, cleanup := newTestServer(t, game)
	defer cleanup()

	ctx := context.Background()
	accName := login_testutil.UniqName("do_tester")

	// 1. 注册（走 Do）
	regBody, _ := proto.Marshal(&pb_account.CreateAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "123456",
	})
	resp, err := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginCreateAccount,
		Message: &pb_common.MessagePacket{Body: regBody},
	})
	if err != nil {
		t.Fatalf("do create account: %v", err)
	}
	if resp.GetMessage().GetErrCode() != pb_error_code.ErrorCode_NoneErr {
		t.Fatalf("create should succeed, err_code=%d", resp.GetMessage().GetErrCode())
	}
	var created pb_account.CreateAccountResp
	if err := proto.Unmarshal(resp.GetMessage().GetBody(), &created); err != nil {
		t.Fatalf("unmarshal create resp: %v", err)
	}
	if created.GetAccountId() == 0 || created.GetToken() == "" {
		t.Fatalf("create resp should have id+token")
	}
	accountID := created.GetAccountId()

	// 2. 重复注册 → AccountExists
	resp2, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginCreateAccount,
		Message: &pb_common.MessagePacket{Body: regBody},
	})
	if resp2.GetMessage().GetErrCode() != pb_error_code.ErrorCode_AccountExists {
		t.Fatalf("duplicate register should be AccountExists, got %d", resp2.GetMessage().GetErrCode())
	}

	// 3. 登录错密码 → AccountOrPasswordWrong
	wrongBody, _ := proto.Marshal(&pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "bad",
	})
	resp3, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginAccount,
		Message: &pb_common.MessagePacket{Body: wrongBody},
	})
	if resp3.GetMessage().GetErrCode() != pb_error_code.ErrorCode_AccountOrPasswordWrong {
		t.Fatalf("wrong password should be AccountOrPasswordWrong, got %d", resp3.GetMessage().GetErrCode())
	}

	// 4. 登录成功拿 token
	loginBody, _ := proto.Marshal(&pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "123456",
	})
	resp4, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginAccount,
		Message: &pb_common.MessagePacket{Body: loginBody},
	})
	if resp4.GetMessage().GetErrCode() != pb_error_code.ErrorCode_NoneErr {
		t.Fatalf("login should succeed, err_code=%d", resp4.GetMessage().GetErrCode())
	}
	var loginOut pb_account.LoginAccountResp
	if err := proto.Unmarshal(resp4.GetMessage().GetBody(), &loginOut); err != nil {
		t.Fatalf("unmarshal login resp: %v", err)
	}

	// 5. EnterServer token 无效 → TokenInvalid
	enterBad, _ := proto.Marshal(&pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: 0, RoleName: login_testutil.UniqName("HeroA"), Token: "bad",
	})
	resp5, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginEnterServer,
		Message: &pb_common.MessagePacket{Body: enterBad},
	})
	if resp5.GetMessage().GetErrCode() != pb_error_code.ErrorCode_TokenInvalid {
		t.Fatalf("bad token should be TokenInvalid, got %d", resp5.GetMessage().GetErrCode())
	}

	// 6. EnterServer 有效 token 新建角 → 成功（game mock 生效）
	enter, _ := proto.Marshal(&pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: 0, RoleName: login_testutil.UniqName("HeroA"), Token: loginOut.GetToken(),
	})
	resp6, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginEnterServer,
		Message: &pb_common.MessagePacket{Body: enter},
	})
	if resp6.GetMessage().GetErrCode() != pb_error_code.ErrorCode_NoneErr {
		t.Fatalf("enter server should succeed, err_code=%d dev=%s", resp6.GetMessage().GetErrCode(), resp6.GetMessage().GetDevMsg())
	}

	// 7. 未知协议 → ProtocolNotFound
	resp7, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_GameHeroList,
		Message: &pb_common.MessagePacket{},
	})
	if resp7.GetMessage().GetErrCode() != pb_error_code.ErrorCode_ProtocolNotFound {
		t.Fatalf("unknown protocol should be ProtocolNotFound, got %d", resp7.GetMessage().GetErrCode())
	}

	// 8. 非法 body → ParamError
	resp8, _ := cli.Do(ctx, &pb_common.NodePacket{
		MsgId:   pb_protocol.MsgID_LoginServerList,
		Message: &pb_common.MessagePacket{Body: []byte("garbage")},
	})
	if resp8.GetMessage().GetErrCode() != pb_error_code.ErrorCode_ParamError {
		t.Fatalf("bad body should be ParamError, got %d", resp8.GetMessage().GetErrCode())
	}
}
