package rpc_conns

import (
	"server.slg.com/api/protocol/pb/pb_gateway"
	"server.slg.com/common/conns/etcdconn"
)

// GetConnByNodeType 通过节点类型获取连接（从 etcd 发现 + 连接池复用）
func GetConnByNodeType(nodeType etcdconn.NodeService) (*NodeConn, error) {
	addr, err := etcdconn.GetNodeTypeServerAddr(nodeType)
	if err != nil {
		return nil, err
	}
	return GetConn(addr)
}

func GetGameServiceNodeCli() (pb_gateway.GatewayNodeServiceClient, error) {
	nodeC, err := GetConnByNodeType(etcdconn.NodeGameService)
	if err != nil {
		return nil, err
	}
	return pb_gateway.NewGatewayNodeServiceClient(nodeC.ClientConn), nil
}
