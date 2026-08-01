package rpc_conns

import (
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/etcdconn"
)

// GetConnByNodeType 通过节点类型获取连接（从 etcd 发现 + 连接池复用）
// 仅按 nodeType 取第一个实例，适用于同类型单实例场景
func GetConnByNodeType(nodeType common_declarations.NodeService) (*NodeConn, error) {
	addr, err := etcdconn.GetNodeTypeServerAddr(nodeType)
	if err != nil {
		return nil, err
	}
	return GetConn(addr)
}

// GetConnByNodeTypeInstance 通过节点类型 + 实例 ID 获取连接（从 etcd 精确发现 + 连接池复用）
//
// 适用于单例配对场景：调用方与目标节点用相同 instance 对齐（如 game 服 ↔ 它的 worldmap）。
func GetConnByNodeTypeInstance(nodeType common_declarations.NodeService, instance string) (*NodeConn, error) {
	addr, err := etcdconn.GetNodeTypeServerAddrByInstance(nodeType, instance)
	if err != nil {
		return nil, err
	}
	return GetConn(addr)
}
