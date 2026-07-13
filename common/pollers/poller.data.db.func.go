package pollers

import (
	"bytes"
	"context"
	"sync/atomic"

	"go.uber.org/zap"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
)

// IsDelete IsDelete
func (p *Poller[M]) IsDelete() bool { return p.data.IsDelete() }

// Saving Saving
func (p *Poller[M]) Saving() *atomic.Bool { return p.saving }

// Save 保存数据
func (p *Poller[M]) Save() {
	if p.data.IsCopy() {
		loggers.Logger.Panic("save copy data", zap.String("tag", p.tag))
	}
	if common_globals.IsTest() {
		return
	}
	// todo async save
	p.manager.asyncSave.Save(p)
}

// SaveSync 立即保存数据
func (p *Poller[M]) SaveSync() {
	if p.data.IsCopy() {
		loggers.Logger.Panic("save copy data", zap.String("tag", p.tag))
	}
	if common_globals.IsTest() {
		return
	}
	p.manager.asyncSave.SaveSync(p)
}

// SaveDo SaveDo
func (p *Poller[M]) SaveDo() {
	data := p.GetSync()
	p.rw.Lock()

	// 1. 获取编码后的数据
	b, err := data.Marshal()
	if err != nil {
		p.rw.Unlock()
		p.Release()

		loggers.Logger.Error(
			"marshal failed",
			zap.String("tag", p.Tag()),
			zap.Uint64("id", data.UniqueID()),
			zap.Error(err),
		)
		return
	}
	p.rw.Unlock()
	p.Release()

	// 2. 与已存储的数据相比，是否有变化
	// 前后必须使用同一个 json 库
	if bytes.Equal(b, data.JSON2Bytes()) {
		return
	}

	// 3. 更新到redis
	cacheKey := p.manager.calcCacheKey(data.UniqueID())
	err = cacheconn.Get().Set(p.manager.ctx, cacheKey, b, p.manager.cacheTTL).Err()
	if err != nil {
		loggers.Logger.Error(
			"cache.Set failed",
			zap.String("tag", p.tag),
			zap.String("cacheKey", cacheKey),
			zap.Uint64("id", data.UniqueID()),
			zap.Error(err),
		)
		return
	}

	data.Bytes2JSON(b)

	// 4. 添加到队列中，会定时取出，然后写入数据库
	err = cacheconn.Get().SAdd(context.Background(), p.manager.cacheQueueKey(), cacheKey).Err()
	if err != nil {
		loggers.Logger.Error(
			"cache.SAdd failed",
			zap.String("tag", p.tag),
			zap.String("syncDBCacheKey", p.manager.cacheQueueKey()),
			zap.String("cacheKey", cacheKey),
			zap.Uint64("id", data.UniqueID()),
			zap.Error(err),
		)
	}
}
