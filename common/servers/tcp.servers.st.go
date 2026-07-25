package servers

import (
	"fmt"
	"net"
	"sync"

	"server.slg.com/common/loggers"
)

// ConnHandler TCP 连接处理函数，由使用者注入具体业务逻辑
type ConnHandler func(conn net.Conn)

// TcpBuilder TCP Server 构造器，只负责构建，不管理生命周期
type TcpBuilder struct {
	config  Config
	handler ConnHandler
}

// NewTcpBuilder 创建 TCP 构造器
func NewTcpBuilder(cfg Config) *TcpBuilder {
	return &TcpBuilder{config: cfg}
}

// WithHandler 注入连接处理函数
func (b *TcpBuilder) WithHandler(h ConnHandler) *TcpBuilder {
	b.handler = h
	return b
}

// Build 构建 TcpServer
func (b *TcpBuilder) Build() *TcpServer {
	return &TcpServer{
		config:  b.config,
		handler: b.handler,
		connMap: make(map[net.Conn]struct{}),
	}
}

// TcpServer TCP 服务器（纯运行时）
// 不管理生命周期，由 Lifecycle 统一协调启动和关闭
type TcpServer struct {
	config  Config
	handler ConnHandler
	lock    sync.Mutex
	connMap map[net.Conn]struct{}
}

// Serve 接受连接循环，阻塞直到 listener 关闭
func (s *TcpServer) Serve(lis net.Listener) {
	loggers.Logger.Info(fmt.Sprintf("TCP 服务开始监听: %s", s.config.Addr))
	for {
		conn, err := lis.Accept()
		if err != nil {
			return
		}
		s.addConn(conn)
		go func() {
			s.handler(conn)
			s.removeConn(conn)
		}()
	}
}

// CloseAll 强制关闭所有活跃连接
func (s *TcpServer) CloseAll() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for conn := range s.connMap {
		conn.Close()
	}
	s.connMap = make(map[net.Conn]struct{})
}

// Len 返回当前活跃连接数
func (s *TcpServer) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.connMap)
}

func (s *TcpServer) addConn(conn net.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.connMap[conn] = struct{}{}
}

func (s *TcpServer) removeConn(conn net.Conn) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.connMap, conn)
}
