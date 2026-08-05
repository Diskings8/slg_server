package game_conf

import (
	"context"
	"os"
	"path/filepath"
	"time"

	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/loggers"
	"go.uber.org/zap"
)

const defaultWatchInterval = 2 * time.Second

// StartWatch 启动配置热更监听（mtime 轮询，无 fsnotify 依赖）。
//
// 必须接收全局 ctx（见 CLAUDE.md 全局 Context 规范）：ctx 取消 → 监听协程退出。
// 检测到目录 mtime 变化 → ReLoad()（内部失败回滚保持旧配置）。
func StartWatch(ctx context.Context, interval time.Duration) {
	path := common_configs.GetConf().GameConf.ConfigPath
	if path == "" {
		loggers.Logger.Warn("game_conf watch skipped: config path not set")
		return
	}
	if interval <= 0 {
		interval = defaultWatchInterval
	}

	go func() {
		last := snapshotMtimes(path)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				loggers.Logger.Info("game_conf watch stopped")
				return
			case <-ticker.C:
				cur := snapshotMtimes(path)
				if mtimesEqual(last, cur) {
					continue
				}
				last = cur
				if err := ReLoad(); err != nil {
					// ReLoad 内部已记日志并保持旧配置
				}
			}
		}
	}()

	loggers.Logger.Info("game_conf watch started", zap.String("path", path), zap.Duration("interval", interval))
}

// snapshotMtimes 采集当前表集（battleOnly 或全量）对应 JSON 文件的 mtime。
func snapshotMtimes(dir string) map[string]time.Time {
	gc := defaultConf.Load()
	regs := allTables
	if gc != nil && gc.battleOnly {
		regs = battleTables
	}
	m := make(map[string]time.Time, len(regs))
	for _, r := range regs {
		fi, err := os.Stat(filepath.Join(dir, r.file+".json"))
		if err == nil {
			m[r.file] = fi.ModTime()
		}
	}
	return m
}

// mtimesEqual 两张 mtime 快照是否一致。
func mtimesEqual(a, b map[string]time.Time) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !bv.Equal(v) {
			return false
		}
	}
	return true
}
