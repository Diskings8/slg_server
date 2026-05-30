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

func registerWithLease(key, value string) {
	resp, err := etcdClient.Grant(context.Background(), 10)
	if err != nil {
		loggers.Log.Warn(fmt.Sprintf("etcd租约失败: %v", err))
		return
	}
	_, err = etcdClient.Put(context.Background(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		loggers.Log.Warn(fmt.Sprintf("etcd注册失败: %v", err))
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
