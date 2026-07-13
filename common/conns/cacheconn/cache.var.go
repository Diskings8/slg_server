package cacheconn

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"server.slg.com/common/configs"
	"server.slg.com/common/utils/util_bytes"
)

var cacheManager CacheI

type CacheI interface {
	SRandMemberN(context.Context, string, int64) *redis.StringSliceCmd
	Expire(background context.Context, key string, ttl time.Duration) *redis.BoolCmd
	Set(background context.Context, key string, b any, ttl time.Duration) *redis.StatusCmd
	SAdd(background context.Context, key string, member ...any) *redis.IntCmd
	Get(background context.Context, key string) *redis.StringCmd
}

// Sep 缓存分隔符
func Sep() byte {
	return ':'
}

func Get() CacheI {
	if cacheManager == nil {
		switch configs.GEnvConf.Redis.GetNodeType() {
		case "single":
			cacheManager = NewCacheSingleManager("")
		default:
			cacheManager = NewCacheSingleManager("")
		}
	}
	return cacheManager
}

func Key(keys ...string) string {
	conf := configs.GEnvConf
	buffer := util_bytes.Get().Buffer(128)
	buffer.WriteString(conf.Redis.GetPrefix())
	buffer.WriteByte(Sep())
	buffer.WriteString(conf.GetNodeType())

	for _, key := range keys {
		buffer.WriteByte(Sep())
		buffer.WriteString(key)
	}

	v := buffer.String()

	util_bytes.Get().Release(buffer)
	return v
}
