package game_conf

import (
	"context"
	"os"
	"time"

	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/loggers"
	"go.uber.org/zap"
)

const defaultWatchInterval = 2 * time.Second

// StartWatch 启动配置热更监听（单文件 mtime+size 轮询，无 fsnotify 依赖）。
//
// 必须接收全局 ctx（见 CLAUDE.md 全局 Context 规范）：ctx 取消 → 监听协程退出。
// 检测到 gameconfig.json 变化 → ReLoad()（内部失败回滚保持旧配置）。
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
		last := snapshotFile(path)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				loggers.Logger.Info("game_conf watch stopped")
				return
			case <-ticker.C:
				cur := snapshotFile(path)
				if sameFileSnapshot(last, cur) {
					continue
				}
				last = cur
				_ = ReLoad() // ReLoad 内部已记日志并保持旧配置
			}
		}
	}()

	loggers.Logger.Info("game_conf watch started", zap.String("path", path), zap.Duration("interval", interval))
}

// fileSnapshot 单文件快照（mtime + size；改内容不碰 mtime 的极端场景靠 size 兜底）。
type fileSnapshot struct {
	modTime time.Time
	size    int64
}

// snapshotFile 采集 gameconfig.json 当前快照（config_path 无效/文件缺失 → 零值，下次轮询再探）。
func snapshotFile(path string) fileSnapshot {
	file, err := resolveGameconfig(path)
	if err != nil {
		return fileSnapshot{}
	}
	fi, err := os.Stat(file)
	if err != nil {
		return fileSnapshot{}
	}
	return fileSnapshot{modTime: fi.ModTime(), size: fi.Size()}
}

// sameFileSnapshot 两次快照是否一致。
func sameFileSnapshot(a, b fileSnapshot) bool {
	return a.size == b.size && a.modTime.Equal(b.modTime)
}
