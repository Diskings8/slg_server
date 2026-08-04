package etcdconn

import (
	"context"
	"fmt"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
)

func GetServiceKeyByNodeType(nodeType common_declarations.NodeService) string {
	switch nodeType {
	case common_declarations.NodeGameService:
		return "/node:service:game/"
	case common_declarations.NodeGatewayService:
		return "/node:service:gateway/"
	case common_declarations.NodeWorldMapService:
		return "/node:service:worldmap/"
	case common_declarations.NodeBattleService:
		return "/node:service:battle/"
	case common_declarations.NodeBattleRecordService:
		return "/node:service:battle_record/"
	default:
		return "/node:service:undef/"
	}
}

// RegisterServiceByNodeType 注册etcd
func RegisterServiceByNodeType(ctx context.Context, nodeType common_declarations.NodeService, instance string, addr string) {
	var key = GetServiceKeyByNodeType(nodeType) + instance + "/"
	registerWithLease(ctx, key, addr)
}

func GetNodeTypeServerList(ctx context.Context, nodeType common_declarations.NodeService) ([]string, error) {
	key := GetServiceKeyByNodeType(nodeType)
	resp, err := etcdClient.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var addrs []string
	for _, kv := range resp.Kvs {
		addrs = append(addrs, string(kv.Key), string(kv.Value))
	}
	return addrs, nil
}

func GetNodeTypeServerAddr(nodeType common_declarations.NodeService) (string, error) {
	key := GetServiceKeyByNodeType(nodeType)
	resp, err := etcdClient.Get(context.Background(), key, clientv3.WithPrefix())
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no available server for node type %d", nodeType)
	}

	return string(resp.Kvs[0].Value), nil
}

// GetNodeTypeServerAddrByInstance 按实例 ID 精确获取节点地址
//
// 适用于单例配对场景：注册 key 为 {prefix}/{instance}/，
// 调用方（如 game 服）用与目标节点相同的 instance 精确定位，避免误连其他实例。
func GetNodeTypeServerAddrByInstance(nodeType common_declarations.NodeService, instance string) (string, error) {
	key := GetServiceKeyByNodeType(nodeType) + instance + "/"
	resp, err := etcdClient.Get(context.Background(), key)
	if err != nil {
		return "", err
	}
	if len(resp.Kvs) == 0 {
		return "", fmt.Errorf("no available server for node type %d instance %s", nodeType, instance)
	}
	return string(resp.Kvs[0].Value), nil
}

func registerWithLease(ctx context.Context, key, value string) {
	resp, err := etcdClient.Grant(ctx, 10)
	if err != nil {
		loggers.Logger.Warn(fmt.Sprintf("etcd租约失败: %v", err))
		return
	}
	ctxTime, cancelFunc := context.WithTimeout(ctx, 3*time.Second)
	defer cancelFunc()
	_, err = etcdClient.Put(ctxTime, key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		loggers.Logger.Warn(fmt.Sprintf("etcd注册失败: %v", err))
		return
	}
	// 自动续租
	ch, err := etcdClient.KeepAlive(ctx, resp.ID)
	if err != nil {
		return
	}
	go func() {
		for range ch {
		}
	}()
}
