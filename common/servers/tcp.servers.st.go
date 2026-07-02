package servers

import (
	"context"
	"fmt"
	"net"
	"sync"

	"server.slg.com/common/conns/netconn"
	"server.slg.com/common/conns/netconn/tcp_conn"
	"server.slg.com/common/loggers"
	"server.slg.com/services/gateway/session_gateways"
)

// TcpServer TCP 服务器，管理 TCP 监听、连接接受和客户端会话生命周期
type TcpServer struct {
	config  Config
	ctx     context.Context
	lock    sync.Mutex
	connMap map[netconn.NetConnI]*session_gateways.Session
}

func BuildTcpServer(ctx context.Context, cfg Config) *TcpServer {
	return &TcpServer{
		ctx:     ctx,
		config:  cfg,
		connMap: make(map[netconn.NetConnI]*session_gateways.Session),
	}
}
func (s *TcpServer) Run() error {
	lis, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return err
	}

	loggers.Logger.Info(fmt.Sprintf("tcp 服务启动成功: %s", s.config.Addr))

	go func() {
		defer func() {
			if e := recover(); e != nil {
				loggers.Logger.Error(fmt.Sprintf("%+v", e))
			}
		}()
		for {
			conn, err := lis.Accept()
			if err != nil {
				return
			}
			go s.handleNewConn(conn)
		}
	}()

	go func() {
		select {
		case <-s.ctx.Done():
			s.gracefulStop()
		}
	}()
	return nil
}

func (s *TcpServer) gracefulStop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, session := range s.connMap {
		session.Close()
	}
}

func (s *TcpServer) addConnData(nc netconn.NetConnI, session *session_gateways.Session) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connMap[nc] = session
}
func (s *TcpServer) removeConnData(session *session_gateways.Session) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.connMap, session.GetConn())
}

func (s *TcpServer) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.connMap)
}

func (s *TcpServer) handleNewConn(conn net.Conn) {
	netc := tcp_conn.NewNetConn(conn)
	session := session_gateways.NewSession(netc)
	s.addConnData(netc, session)
	session.RunToReceiveFromConn()
	// 断开
	s.removeConnData(session)
}
