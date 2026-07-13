package pollers

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/asyncsave_entity"
	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/common/utils/util_randoms"
)

type PollerManager[M common_declarations.DataI] struct {
	ctx context.Context
	// all: 所有轮询器
	// key: 唯一id
	// value: 轮询器
	all hashmaps.Map[uint64, *Poller[M]]

	// lockers: id对应的锁
	// key: 唯一id
	// value: sync.Mutex
	lockers hashmaps.Map[uint64, *sync.Mutex]

	// loaderFunc 数据加载器,当本地数据和缓存数据都不存在时，加载数据
	loaderFunc common_declarations.LoaderFunc[M]

	// cacheTTL 远端缓存时长
	cacheTTL time.Duration

	//
	newFunc func() M

	//
	cacheQueueKey func() string

	//
	zeroVal M

	//
	asyncSave     *asyncsave_entity.AsyncSaveEntity
	dbCronEntryID cron.EntryID
	//
	closeChan chan struct{}
}

func (p *PollerManager[M]) makeCacheKey(id uint64) string {
	return strconv.FormatUint(id, 10)
}

func (p *PollerManager[M]) calcCacheKey(id uint64) string {
	return cacheconn.Key(common_declarations.CaCheKEY, p.zeroVal.Tag(), p.makeCacheKey(id))
}

func (p *PollerManager[M]) cleanup() {
	ticker := time.NewTicker(common_declarations.PollerCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().Unix()
			p.all.Range(func(key uint64, value *Poller[M]) bool {
				poller := value
				// 上次活跃已经超过最大失效
				if now-poller.activeAt.Load() > int64(common_declarations.PollerInactiveTimeout.Seconds()) {
					poller.Save()

					p.all.Delete(key)
					p.lockers.Delete(key)

					// 保存到数据库
					err := poller.data.Save(poller.data.IsDelete())
					if err != nil {
						loggers.Logger.Error(err.Error())
					}
				}
				return true
			})
		case <-p.closeChan:
			return
		}
	}
}

func (p *PollerManager[M]) saveToDB() {
	cache := cacheconn.Get()

	keys, err := cache.SRandMemberN(context.Background(), p.cacheQueueKey(), 200).Result()
	if err != nil {
		loggers.Logger.Error("cache.SRandMemberN failed", zap.String("cacheQueueKey", p.cacheQueueKey()), zap.Error(err))
		return
	}

	for _, key := range keys {
		err := p.syncToDB(key)
		if err != nil {
			loggers.Logger.Error("sync to db failed", zap.String("cacheQueueKey", p.cacheQueueKey()), zap.String("key", key), zap.Error(err))
			continue
		}

		// 重新设置缓存 TTL
		if p.cacheTTL > 0 {
			ttl := p.cacheTTL + time.Duration(util_randoms.BetweenInt64(30, 120))*time.Second
			cache.Expire(context.Background(), key, ttl)
		}
	}
}

func (p *PollerManager[M]) syncToDB(key string) error {
	return nil
}

func (p *PollerManager[M]) load(id uint64) (M, error) {
	// 1. 从缓存中获取
	if v, err := p.loadFromCache(id); err == nil {
		return v, nil
	}

	// 2. 从加载器中获取
	v, err := p.loaderFunc(id)
	if err != nil {
		return p.zeroVal, err
	}

	// 3. 保存到缓存中
	if err := p.saveToCache(id, v); err != nil {
		return p.zeroVal, err
	}

	return v, nil
}

func (p *PollerManager[M]) loadFromCache(id uint64) (M, error) {
	cacheKey := p.calcCacheKey(id)

	b, err := cacheconn.Get().Get(p.ctx, cacheKey).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			loggers.Logger.Error("[pkg.pollers.load] failed", zap.String("key", cacheKey), zap.Error(err))
		}
		return p.zeroVal, err
	}

	m := p.newFunc()
	if err := m.Unmarshal(b); err != nil {
		loggers.Logger.Error("[pkg.pollers.load] failed", zap.String("key", cacheKey), zap.Error(err))
		return p.zeroVal, err
	}

	return m, nil
}

func (p *PollerManager[M]) saveToCache(id uint64, data M) error {
	cacheKey := p.calcCacheKey(id)

	b, err := data.Marshal()
	if err != nil {
		loggers.Logger.Error("[pkg.pollers.save] failed", zap.String("key", cacheKey), zap.Error(err))
		return err
	}

	if err := cacheconn.Get().Set(context.Background(), cacheKey, b, p.cacheTTL).Err(); err != nil {
		loggers.Logger.Error("[pkg.pollers.save] failed", zap.String("key", cacheKey), zap.Error(err))
		return err
	}

	return nil
}
