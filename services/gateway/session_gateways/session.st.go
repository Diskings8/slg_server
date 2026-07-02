package session_gateways

import (
	"fmt"

	"server.slg.com/common/conns/netconn"
	"server.slg.com/common/conns/netconn/packets"
	"server.slg.com/common/loggers"
)

type Session struct {
	conn   netconn.NetConnI
	closed bool
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
			loggers.Log.Error(fmt.Sprintf("session RunToReceiveFromConn error :%+v", e))
		}
	}()
	for {
		packet, err := s.conn.ReadFromConn()
		if err != nil {
			loggers.Log.Info(fmt.Sprintf("客户端断开: %v", err))
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
			loggers.Log.Error(fmt.Sprintf("session RunToSendToConn error :%+v", e))
		}
	}()
	// todo 收到来自game 服务的stream链接信息
}

func (s *Session) switchForward(packet *packets.Packet) {
	// todo 根据 message id 划分路由
}
