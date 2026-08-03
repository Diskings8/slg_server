package rpc_conns

import (
	"context"
	"fmt"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"server.slg.com/common/conns/rpcconn/rpc_declarations"
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

// GetConn 通过地址获取连接（内置默认超时，连接池管理）
func GetConn(addr string) (*NodeConn, error) {
	return defaultPool.getOrCreateConn(addr, rpc_declarations.DefaultRPCTimeout)
}

// GetConnWithTimeout 通过地址获取连接（指定超时；timeout<=0 时无限等待）
func GetConnWithTimeout(addr string, timeout time.Duration) (*NodeConn, error) {
	return defaultPool.getOrCreateConn(addr, timeout)
}

// GetConnWait 通过地址获取连接（阻塞等待就绪，无超时）
func GetConnWait(addr string) (*NodeConn, error) {
	return defaultPool.getOrCreateConn(addr, 0)
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

func (p *nodeConnPool) getOrCreateConn(addr string, timeout time.Duration) (*NodeConn, error) {
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

	conn, err := dial(addr, timeout)
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

// dial 建立连接；timeout>0 时等待就绪超时失败，timeout<=0 时阻塞等待就绪（不超时）
func dial(addr string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx := context.Background()
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

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
