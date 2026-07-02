package etcdconn

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
	"server.slg.com/common/loggers"
)

func GetServiceKeyByNodeType(nodeType NodeService) string {
	switch nodeType {
	case NodeGameService:
		return "/node:service:game/"
	case NodeGatewayService:
		return "/node:service:gateway/"
	default:
		return "/node:service:undef/"
	}
}

// RegisterServiceByNodeType 注册etcd
func RegisterServiceByNodeType(nodeType NodeService, instance string, addr string) {
	var key = GetServiceKeyByNodeType(nodeType) + instance + "/"
	registerWithLease(key, addr)
}

func GetNodeTypeServerList(nodeType NodeService) ([]string, error) {
	key := GetServiceKeyByNodeType(nodeType)
	resp, err := etcdClient.Get(context.Background(), key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Key), string(kv.Value))
	}
	return addrs, nil
}

func GetNodeTypeServerAddr(nodeType NodeService) (string, error) {
	key := GetServiceKeyByNodeType(nodeType)
	resp, err := etcdClient.Get(context.Background(), key, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no available server for node type %d", nodeType)
	}

	// etcd key: /node:service:game/{instance}/
	// etcd value: {addr}
	return string(resp.Kvs[0].Value), nil
}

func registerWithLease(key, value string) {
	resp, err := etcdClient.Grant(context.Background(), 10)
	if err != nil {
		loggers.Logger.Warn(fmt.Sprintf("etcd租约失败: %v", err))
		return
	}
	_, err = etcdClient.Put(context.Background(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		loggers.Logger.Warn(fmt.Sprintf("etcd注册失败: %v", err))
		return
	}
	// 自动续租
	ch, err := etcdClient.KeepAlive(context.Background(), resp.ID)
	if err != nil {
		return
	}
	go func() {
		for range ch {
		}
	}()
}
