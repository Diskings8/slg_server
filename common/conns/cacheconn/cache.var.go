package cacheconn

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/utils/util_bytes"
)

var cacheManager CacheI

type CacheI interface {
	SRandMemberN(context.Context, string, int64) *redis.StringSliceCmd
	Expire(background context.Context, key string, ttl time.Duration) *redis.BoolCmd
	Set(background context.Context, key string, b any, ttl time.Duration) *redis.StatusCmd
	SAdd(background context.Context, key string, member ...any) *redis.IntCmd
	SRem(background context.Context, key string, members ...any) *redis.IntCmd // poller 写库成功后从脏队列移除
	Get(background context.Context, key string) *redis.StringCmd
	SetNX(background context.Context, key string, value any, ttl time.Duration) *redis.BoolCmd // 幂等标记（不存在才设置）

	// Pub/Sub：login 发布进服广播，gateway 订阅踢旧连接
	Publish(ctx context.Context, channel string, msg any) *redis.IntCmd
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

// Sep 缓存分隔符
func Sep() byte {
	return ':'
}

// Init 初始化 redis 连接（single / cluster），启动时由服务 main.go 调用。
// Ping 验证连接，失败快速返回（fail fast）。
func Init(ctx context.Context) error {
	initManager()
	switch m := cacheManager.(type) {
	case *CacheSingleManager:
		return m.Ping(ctx).Err()
	case *CacheClusterManager:
		return m.Ping(ctx).Err()
	}
	return nil
}

// ShutDown 关闭 redis 连接（进程优雅退出）
func ShutDown() error {
	if cacheManager == nil {
		return nil
	}
	if cli, ok := any(cacheManager).(interface{ Close() error }); ok {
		return cli.Close()
	}
	return nil
}

// initManager 按配置建立 redis client（single / cluster），幂等。
// Address 为空时使用默认 localhost:6379（go-redis），连接失败在调用方报错而非 panic。
func initManager() {
	if cacheManager != nil {
		return
	}
	conf := common_configs.GetConf().Cache
	switch conf.GetNodeType() {
	case "cluster":
		cacheManager = &CacheClusterManager{
			ClusterClient: redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:    conf.Address,
				Password: conf.Password,
			}),
		}
	default: // single
		addr := "localhost:6379"
		if len(conf.Address) > 0 && conf.Address[0] != "" {
			addr = conf.Address[0]
		}
		cacheManager = &CacheSingleManager{
			Client: redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: conf.Password,
				DB:       conf.DB,
			}),
		}
	}
}

func Get() CacheI {
	if cacheManager == nil {
		initManager()
	}
	return cacheManager
}

func Key(keys ...string) string {
	conf := common_configs.GetConf()
	buffer := util_bytes.Get().Buffer(128)
	buffer.WriteString(conf.Cache.GetPrefix())
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
