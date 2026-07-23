package rpc_conns

import (
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/etcdconn"
)

// GetConnByNodeType 通过节点类型获取连接（从 etcd 发现 + 连接池复用）
func GetConnByNodeType(nodeType common_declarations.NodeService) (*NodeConn, error) {
	addr, err := etcdconn.GetNodeTypeServerAddr(nodeType)
	if err != nil {
		return nil, err
	}
	return GetConn(addr)
}
