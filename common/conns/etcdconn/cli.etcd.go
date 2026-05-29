package etcdconn

import (
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitEtcd(dsn string) {
	once.Do(func() {
		cli, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{dsn},
			DialTimeout: 3 * time.Second,
		})
		if err != nil {
			log.Fatal("etcd 连接失败:", err)
		}
		etcdClient = cli
	})
}

func GetCli() *clientv3.Client {
	return etcdClient
}
