package session_gateways

import (
	"context"
	"fmt"
	"sync"

	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/conns/netconn"
	"server.slg.com/common/conns/netconn/packets"
	"server.slg.com/common/loggers"
)

// nodeID 当前 gateway 进程唯一标识（main.go SetNodeID 设置）。
// 进服转发时填 MessagePacket.GatewayNodeId，login 据此记录/广播角色所在 gateway。
var nodeID string

// SetNodeID 设置当前 gateway 节点标识（main.go 启动时调用）
func SetNodeID(id string) { nodeID = id }

// Session 客户端会话，持有一个网络连接并提供数据收发和转发能力。
//
// 进服（EnterServer 成功）后维护一条到 game 的双向流（gameStream），上行转发请求、下行回包；
// 同时登记到 defaultSessionManager（roleID → Session），供下推 RPC 按角色定位连接。
//
// ctx 为全局 context（main.go 传入）：game 双向流、出向 RPC 均从它派生，
// 服务关停时 ctx.Done() 直接传导到这些长驻资源，而非依赖 TCP 断开的间接级联。
type Session struct {
	ctx       context.Context
	conn      netconn.NetConnI
	closed    bool
	roleID    uint64
	accountID uint64
	serverID  uint32

	streamMu     sync.Mutex                    // 保护 gameStream / streamCancel / lastSeq
	gameStream   pb_game.GameService_StreamClient // 当前活跃的 game 双向流（nil = 未进服/已断开）
	streamCancel context.CancelFunc
	lastSeq      uint32 // 最近一次上行请求的 seq（下行响应回 seq 用）

	writeMu sync.Mutex // 串行化客户端 TCP 写入（read-loop / recvFromGame / 下推 RPC 多 goroutine 并发写）
}

// NewSession 创建会话，ctx 为全局 context（长驻资源派生源；nil 时退化为 Background）
func NewSession(ctx context.Context, conn netconn.NetConnI) *Session {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Session{ctx: ctx, conn: conn}
}

func (s *Session) GetConn() netconn.NetConnI {
	return s.conn
}

func (s *Session) Close() {
	_ = s.conn.Close()
}

func (s *Session) RunToReceiveFromConn() {
	defer func() {
		if e := recover(); e != nil {
			loggers.Logger.Error(fmt.Sprintf("session RunToReceiveFromConn error :%+v", e))
		}
		// TCP 断开 → 关闭 game 流（game 侧 gateConnectDo WaitDone 返回 → Offline 下线）
		s.cleanupGameStream()
		// 从下推注册表注销（幂等）
		defaultSessionManager.Unregister(s.roleID)
	}()
	for {
		packet, err := s.conn.ReadFromConn()
		if err != nil {
			loggers.Logger.Info(fmt.Sprintf("客户端断开: %v", err))
			return
		}
		//
		s.switchForward(packet)
		packet.Release()
	}
}

// writePacket 串行化客户端 TCP 写入（read-loop / recvFromGame / 下推 RPC 多 goroutine 并发写，需互斥）
func (s *Session) writePacket(seq uint32, p *packets.Packet) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteToConn(seq, p)
}
