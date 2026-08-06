package session_gateways

// switchForward 单测 — 用 fake conn 捕获回包 + mock login 客户端注入，验证转发/回包信封/错误码。
// （不依赖 etcd/redis；完整链路需运行时环境验证。）

import (
	"context"
	"io"
	"os"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/api/protocol/pb/pb_common"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_protocol"
	"server.slg.com/common/conns/netconn/packets"
	"server.slg.com/common/loggers"
	"server.slg.com/services/gateway/gateway_internals/gateway_rpc_clients"
)

// TestMain 初始化日志（switchForward 中会写 loggers.Logger）
func TestMain(m *testing.M) {
	loggers.Init()
	os.Exit(m.Run())
}

type written struct {
	seq uint32
	p   *packets.Packet
}

// fakeConn 捕获写入的数据包
type fakeConn struct {
	writes []written
}

func (f *fakeConn) Close() error                      { return nil }
func (f *fakeConn) ReadFromConn() (*packets.Packet, error) { return nil, io.EOF }
func (f *fakeConn) WriteToConn(seq uint32, p *packets.Packet) error {
	f.writes = append(f.writes, written{seq: seq, p: p})
	return nil
}
func (f *fakeConn) RemoteAddr() string { return "test" }

// fakeLoginClient mock login 节点：Do 返回预置响应，其余方法留空
type fakeLoginClient struct {
	doResp *pb_common.NodePacket
	doErr  error
}

func (f *fakeLoginClient) CreateAccount(context.Context, *pb_account.CreateAccountReq, ...grpc.CallOption) (*pb_account.CreateAccountResp, error) {
	return nil, nil
}
func (f *fakeLoginClient) LoginAccount(context.Context, *pb_account.LoginAccountReq, ...grpc.CallOption) (*pb_account.LoginAccountResp, error) {
	return nil, nil
}
func (f *fakeLoginClient) ServerList(context.Context, *pb_account.ServerListReq, ...grpc.CallOption) (*pb_account.ServerListResp, error) {
	return nil, nil
}
func (f *fakeLoginClient) EnterServer(context.Context, *pb_account.EnterServerReq, ...grpc.CallOption) (*pb_account.EnterServerResp, error) {
	return nil, nil
}
func (f *fakeLoginClient) Do(_ context.Context, _ *pb_common.NodePacket, _ ...grpc.CallOption) (*pb_common.NodePacket, error) {
	return f.doResp, f.doErr
}

func TestIsLoginMsgID(t *testing.T) {
	if !isLoginMsgID(pb_protocol.MsgID_LoginAccount) || !isLoginMsgID(pb_protocol.MsgID_LoginCreateAccount) ||
		!isLoginMsgID(pb_protocol.MsgID_LoginServerList) || !isLoginMsgID(pb_protocol.MsgID_LoginEnterServer) {
		t.Fatal("all login msg ids should be classified as login")
	}
	if isLoginMsgID(pb_protocol.MsgID_GameHeroList) {
		t.Fatal("game msg id should not be classified as login")
	}
}

func TestSwitchForwardLogin(t *testing.T) {
	// 注入 mock login 客户端（返回成功响应）
	cli := &fakeLoginClient{
		doResp: &pb_common.NodePacket{
			MsgId: pb_protocol.MsgID_LoginAccount,
			Message: &pb_common.MessagePacket{
				Body: []byte("resp-body"),
			},
		},
	}
	gateway_rpc_clients.Client().SetLoginClient(cli)

	conn := &fakeConn{}
	s := NewSession(conn)

	// 构造登录请求包：LoginAccount MsgID + LoginAccountReq 序列化 body
	reqBytes, _ := proto.Marshal(&pb_account.LoginAccountReq{
		ChannelType: pb_account.ChannelType_Mine,
		AccountName: "tester",
		Password:    "123456",
	})
	s.switchForward(&packets.Packet{Seq: 7, MsgID: uint32(pb_protocol.MsgID_LoginAccount), Body: reqBytes})

	if len(conn.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(conn.writes))
	}
	got := conn.writes[0]
	if got.seq != 7 {
		t.Fatalf("seq should echo request seq: got %d", got.seq)
	}
	if got.p.MsgID != uint32(pb_protocol.MsgID_LoginAccount) {
		t.Fatalf("msg id echo mismatch: %d", got.p.MsgID)
	}
	// 回包 body 应为序列化的 MessagePacket 信封
	var msg pb_common.MessagePacket
	if err := proto.Unmarshal(got.p.Body, &msg); err != nil {
		t.Fatalf("decode response envelope: %v", err)
	}
	if msg.GetErrCode() != pb_error_code.ErrorCode_NoneErr || string(msg.GetBody()) != "resp-body" {
		t.Fatalf("unexpected envelope: err=%d body=%q", msg.GetErrCode(), msg.GetBody())
	}
}

func TestSwitchForwardUnsupported(t *testing.T) {
	conn := &fakeConn{}
	s := NewSession(conn)

	// 非 login 协议（本轮未支持）→ ProtocolNotFound 错误包
	s.switchForward(&packets.Packet{Seq: 1, MsgID: uint32(pb_protocol.MsgID_GameHeroList), Body: nil})

	if len(conn.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(conn.writes))
	}
	var msg pb_common.MessagePacket
	if err := proto.Unmarshal(conn.writes[0].p.Body, &msg); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if msg.GetErrCode() != pb_error_code.ErrorCode_ProtocolNotFound {
		t.Fatalf("unsupported protocol should be ProtocolNotFound, got %d", msg.GetErrCode())
	}
}

func TestSwitchForwardLoginNodeDown(t *testing.T) {
	// 注入一个报错的 login 客户端（模拟节点不可达）
	gateway_rpc_clients.Client().SetLoginClient(&fakeLoginClient{
		doErr: io.EOF,
	})

	conn := &fakeConn{}
	s := NewSession(conn)

	s.switchForward(&packets.Packet{Seq: 2, MsgID: uint32(pb_protocol.MsgID_LoginServerList), Body: nil})

	if len(conn.writes) != 1 {
		t.Fatalf("expected 1 write, got %d", len(conn.writes))
	}
	var msg pb_common.MessagePacket
	if err := proto.Unmarshal(conn.writes[0].p.Body, &msg); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if msg.GetErrCode() != pb_error_code.ErrorCode_SystemBusy {
		t.Fatalf("node down should be SystemBusy, got %d", msg.GetErrCode())
	}
}
