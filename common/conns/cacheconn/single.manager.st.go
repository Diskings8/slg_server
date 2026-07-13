package cacheconn

import (
	"github.com/redis/go-redis/v9"
)

var _ CacheI = new(CacheSingleManager)

type CacheSingleManager struct {
	*redis.Client
	redisBaseKey string
}

func NewCacheSingleManager(rbk string) *CacheSingleManager {
	return &CacheSingleManager{redisBaseKey: rbk}
}
