package cacheconn

type CacheI interface {
	Key(keys ...string) string
}
