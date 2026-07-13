package cacheconn

import (
	"github.com/redis/go-redis/v9"
)

var _ CacheI = new(CacheClusterManager)

type CacheClusterManager struct {
	*redis.ClusterClient
	redisBaseKey string
}

func NewCacheClusterManager(rbk string) *CacheClusterManager {
	return &CacheClusterManager{redisBaseKey: rbk}
}
