package etcdconn

import (
	clientv3 "go.etcd.io/etcd/client/v3"
)

var etcdClient *clientv3.Client
var once sync.Once
