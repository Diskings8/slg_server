package rpcconn

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

// NodeConn gRPC 节点连接，封装了目标地址和 gRPC 客户端连接，通过 Alive 标记连接可用状态
type NodeConn struct {
	Addr string
	*grpc.ClientConn
	Alive bool
}

func (c *NodeConn) Close() error {
	return c.ClientConn.Close()
}

// GetConn 通过地址获取连接（连接池管理）
func GetConn(addr string) (*NodeConn, error) {
	return defaultPool.getOrCreateConn(addr)
}

// CloseAll 关闭连接池中所有连接
func CloseAll() {
	defaultPool.closeAll()
}

var defaultPool = newNodeConnPool()

// nodeConnPool gRPC 节点连接池，提供按地址的连接复用和统一生命周期管理
type nodeConnPool struct {
	rwLock sync.RWMutex
	pool   map[string]*NodeConn
}

func newNodeConnPool() *nodeConnPool {
	return &nodeConnPool{
		pool: make(map[string]*NodeConn),
	}
}

func (p *nodeConnPool) getOrCreateConn(addr string) (*NodeConn, error) {
	// 读锁快速路径
	p.rwLock.RLock()
	nc, ok := p.pool[addr]
	p.rwLock.RUnlock()
	if ok && nc.Alive {
		return nc, nil
	}

	// 写锁 + 双重检查
	p.rwLock.Lock()
	defer p.rwLock.Unlock()

	if nc, ok := p.pool[addr]; ok && nc.Alive {
		return nc, nil
	}

	conn, err := dial(addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}

	nc = &NodeConn{
		Addr:       addr,
		ClientConn: conn,
		Alive:      true,
	}
	p.pool[addr] = nc
	return nc, nil
}

func (p *nodeConnPool) closeAll() {
	p.rwLock.Lock()
	defer p.rwLock.Unlock()
	for addr, nc := range p.pool {
		_ = nc.Close()
		delete(p.pool, addr)
	}
}

func dial(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second, // 每 30s 发一次 ping
			Timeout:             5 * time.Second,  // ping 超时 5s
			PermitWithoutStream: true,             // 无活跃流时也发 ping
		}),
	)
	if err != nil {
		return nil, err
	}

	// NewClient 默认非阻塞，手动等待连接就绪
	for {
		state := conn.GetState()
		if state == connectivity.Ready {
			break
		}
		if !conn.WaitForStateChange(ctx, state) {
			_ = conn.Close()
			return nil, fmt.Errorf("dial %s: timeout", addr)
		}
	}
	return conn, nil
}
