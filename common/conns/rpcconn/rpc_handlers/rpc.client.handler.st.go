package rpc_handlers

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/rpcconn/rpc_conns"
)

// dialNodeLocked 按节点类型 + 实例 ID 获取 gRPC 连接。
// 已建立且就绪则复用；连接不存在或已断开则重新从 etcd 发现（instance 配对）+ 连接池创建。
// wait=true 时阻塞等待就绪（无超时）；否则使用内置默认超时。
// 调用前须持有 ch.mu。
func (ch *ClientHandler) dialNodeLocked(nodeType common_declarations.NodeService, wait bool) (*grpc.ClientConn, error) {
	if conn, ok := ch.conns[nodeType]; ok && conn.GetState() == connectivity.Ready {
		return conn, nil
	}

	var (
		nc  *rpc_conns.NodeConn
		err error
	)
	if wait {
		nc, err = rpc_conns.GetConnByNodeTypeInstanceWait(nodeType, ch.instance)
	} else {
		nc, err = rpc_conns.GetConnByNodeTypeInstance(nodeType, ch.instance)
	}
	if err != nil {
		return nil, err
	}
	ch.conns[nodeType] = nc.ClientConn
	ch.closers = append(ch.closers, nc.ClientConn)
	return nc.ClientConn, nil
}

// Close 关闭 ClientHandler 管理的所有 gRPC 连接。
// Close 后 ClientHandler 不应再被使用，应创建新实例。
func (ch *ClientHandler) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	for _, c := range ch.closers {
		_ = c.Close()
	}
	ch.conns = nil
	ch.closers = nil
}
