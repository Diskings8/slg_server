//go:build integration

package login_servers_test

// 进程内 gRPC 冒烟 — 连真实 mysql（common_db_0），game 建角用 mock。
// 运行：go test -tags integration ./services/login/...
// 覆盖 CreateAccount → LoginAccount → ServerList → EnterServer(新建/已有/越权/game失败)
// → 多渠道绑定（第三渠道登录自动绑定到同一账号、共享角色）全链路。

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_handlers/login_servers"
	"server.slg.com/services/login/login_internals/login_tokens"
	"server.slg.com/services/login/login_testutil"
)

// fakeGameClient mock 建角：可注入失败
type fakeGameClient struct {
	calls    int
	failNext bool
}

func (f *fakeGameClient) CreateRole(_ context.Context, _ uint32, req *pb_common.CreateRoleReq) (*pb_common.CreateRoleRsp, error) {
	f.calls++
	if f.failNext {
		return nil, errors.New("game unavailable")
	}
	return &pb_common.CreateRoleRsp{RoleId: req.GetRoleId()}, nil
}

func newTestServer(t *testing.T, game *fakeGameClient) (pb_account.AccountServiceClient, func()) {
	t.Helper()

	loggers.Init()
	snowflakes.Init()

	accStore, chStore, svStore := login_testutil.SetupStores(t)
	login_servers.LoginServerHandler.SetStore(accStore, chStore, svStore, login_tokens.NewTokenManager(), game)

	lis := bufconn.Listen(1024 * 1024)
	srv := grpc.NewServer()
	pb_account.RegisterAccountServiceServer(srv, login_servers.LoginServerHandler)
	go func() { _ = srv.Serve(lis) }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	if err != nil {
		cancel()
		t.Fatalf("dial: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		srv.Stop()
		cancel()
	}
	return pb_account.NewAccountServiceClient(conn), cleanup
}

func TestLoginFlow(t *testing.T) {
	game := &fakeGameClient{}
	cli, cleanup := newTestServer(t, game)
	defer cleanup()

	ctx := context.Background()
	accName := login_testutil.UniqName("tester")

	// 1. 注册（官方渠道）
	create, err := cli.CreateAccount(ctx, &pb_account.CreateAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if create.GetAccountId() == 0 || create.GetToken() == "" {
		t.Fatalf("create account should return id and token")
	}
	accountID := create.GetAccountId()

	// 2. 重复注册（同名任意渠道）→ AlreadyExists
	if _, err := cli.CreateAccount(ctx, &pb_account.CreateAccountReq{
		ChannelType: pb_account.ChannelType_ThirdParty,
		AccountName: accName,
		Password:    "123456",
	}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate register should be AlreadyExists, got %v", err)
	}

	// 3. 登录（错密码 → Unauthenticated）
	if _, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "wrong",
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("wrong password should be Unauthenticated, got %v", err)
	}

	login, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.GetAccountId() != accountID || len(login.GetRoleList().GetSimpleInfo()) != 0 {
		t.Fatalf("login should return empty role list")
	}
	token := login.GetToken()

	// 4. 区服列表（种子 S1）
	servers, err := cli.ServerList(ctx, &pb_account.ServerListReq{})
	if err != nil {
		t.Fatalf("server list: %v", err)
	}
	foundS1 := false
	for _, sv := range servers.GetServers() {
		if sv.GetServerId() == 1 {
			foundS1 = true
		}
	}
	if !foundS1 {
		t.Fatalf("server list should contain S1: %+v", servers)
	}

	// 5. 进入区服 - 新建角（token 无效 → Unauthenticated）
	if _, err := cli.EnterServer(ctx, &pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: 0, RoleName: "HeroA", Token: "bad",
	}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("bad token should be Unauthenticated, got %v", err)
	}

	enter, err := cli.EnterServer(ctx, &pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: 0, RoleName: login_testutil.UniqName("HeroA"), Token: token,
	})
	if err != nil {
		t.Fatalf("enter server new role: %v", err)
	}
	if enter.GetRoleId() == 0 || game.calls != 1 {
		t.Fatalf("enter server should create role via game: role=%d calls=%d", enter.GetRoleId(), game.calls)
	}
	roleID := enter.GetRoleId()

	// 6. 建角后登录：角色列表含该角色，last_use 已填充
	login2, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("login after create role: %v", err)
	}
	rl := login2.GetRoleList()
	if len(rl.GetSimpleInfo()) != 1 || rl.GetSimpleInfo()[0].GetRoleId() != roleID {
		t.Fatalf("role list should contain created role")
	}
	if rl.GetLastUse() == nil || rl.GetLastUse().GetRoleId() != roleID {
		t.Fatalf("last_use should be the entered role")
	}
	token = login2.GetToken() // 登录重新签发票据

	// 7. 进入已有角色（正确 → 成功；错误 → PermissionDenied）
	if _, err := cli.EnterServer(ctx, &pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: roleID, Token: token,
	}); err != nil {
		t.Fatalf("enter existing role: %v", err)
	}
	if _, err := cli.EnterServer(ctx, &pb_account.EnterServerReq{
		AccountId: accountID, ServerId: 1, RoleId: 99999, Token: token,
	}); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("enter unowned role should be PermissionDenied, got %v", err)
	}

	// 8. 多渠道绑定：第三方渠道登录同一账号名+密码 → 自动绑定 → 同一账号、共享角色
	cross, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_ThirdParty,
		AccountName: accName,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("cross channel login: %v", err)
	}
	if cross.GetAccountId() != accountID {
		t.Fatalf("cross channel should resolve to same account")
	}
	if len(cross.GetRoleList().GetSimpleInfo()) != 1 || cross.GetRoleList().GetSimpleInfo()[0].GetRoleId() != roleID {
		t.Fatalf("cross channel should share roles")
	}

	// 9. game 建角失败：返回错误且不产生映射（用新账号验证）
	accName2 := login_testutil.UniqName("tester2")
	create2, err := cli.CreateAccount(ctx, &pb_account.CreateAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName2,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("create account 2: %v", err)
	}
	login2b, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName2,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("login account 2: %v", err)
	}
	token2 := login2b.GetToken()

	game.failNext = true
	if _, err := cli.EnterServer(ctx, &pb_account.EnterServerReq{
		AccountId: create2.GetAccountId(), ServerId: 1, RoleId: 0, RoleName: login_testutil.UniqName("HeroB"), Token: token2,
	}); status.Code(err) != codes.Internal {
		t.Fatalf("game failure should be Internal, got %v", err)
	}
	login3, err := cli.LoginAccount(ctx, &pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: accName2,
		Password:    "123456",
	})
	if err != nil {
		t.Fatalf("login after failed create: %v", err)
	}
	if len(login3.GetRoleList().GetSimpleInfo()) != 0 {
		t.Fatalf("failed create should leave no mapping")
	}
}
