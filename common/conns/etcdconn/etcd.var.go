package etcdconn

import (
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client
var once sync.Once

type NodeService int

const (
	NodeGameService    NodeService = 10
	NodeGatewayService NodeService = 20
)
