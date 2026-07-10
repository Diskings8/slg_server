package rpc_handlers

import (
	"google.golang.org/grpc"
	"server.slg.com/common/conns/rpcconn/rpc_conns"

	"server.slg.com/common/conns/etcdconn"
)

var RpcClient = &ClientHandler{
	conns: make(map[etcdconn.NodeService]*grpc.ClientConn),
}

// dialNodeLocked 按节点类型获取 gRPC 连接（已建立则复用，未建立则通过 etcd 发现 + 连接池创建）。
// 调用前须持有 ch.mu。
func (ch *ClientHandler) dialNodeLocked(nodeType etcdconn.NodeService) (*grpc.ClientConn, error) {
	if conn, ok := ch.conns[nodeType]; ok {
		return conn, nil
	}
	nc, err := rpc_conns.GetConnByNodeType(nodeType)
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
