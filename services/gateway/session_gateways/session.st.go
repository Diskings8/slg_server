package session_gateways

import (
	"context"
	"fmt"
	"sync"

	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/conns/netconn"
	"server.slg.com/common/loggers"
)

// Session 客户端会话，持有一个网络连接并提供数据收发和转发能力。
//
// 进服（EnterServer 成功）后维护一条到 game 的双向流（gameStream），上行转发请求、下行回包。
type Session struct {
	conn      netconn.NetConnI
	closed    bool
	roleID    uint64
	accountID uint64
	serverID  uint32

	streamMu     sync.Mutex                    // 保护 gameStream / streamCancel / lastSeq
	gameStream   pb_game.GameService_StreamClient // 当前活跃的 game 双向流（nil = 未进服/已断开）
	streamCancel context.CancelFunc
	lastSeq      uint32 // 最近一次上行请求的 seq（下行响应回 seq 用）
}

func NewSession(conn netconn.NetConnI) *Session {
	session := Session{conn: conn}
	return &session
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

func (s *Session) RunToSendToConn() {
	defer func() {
		if e := recover(); e != nil {
			loggers.Logger.Error(fmt.Sprintf("session RunToSendToConn error :%+v", e))
		}
	}()
	// todo 收到来自game 服务的stream链接信息
}
