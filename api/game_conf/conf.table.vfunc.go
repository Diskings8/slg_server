package game_conf

import (
	"sync"
	"sync/atomic"

	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/loggers"
	"go.uber.org/zap"
)

var (
	defaultConf atomic.Pointer[GameConf]
	// reloadMu 串行化所有写者（Init/InitDefault/InitBattle/ReLoad/热更触发），杜绝并发 swap 竞态。
	// 读者无锁：atomic.Pointer.Load() 拿到一致快照。
	reloadMu sync.Mutex
)

// init 兜底：确保默认配置可用（Load 永不返回 nil），供测试/未配置 JSON 环境使用。
func init() {
	_ = InitDefault()
}

// InitFromConf 使用 EnvConf 配置里的 game_conf.config_path 初始化（path 为空走 Go 内嵌）。
func InitFromConf() error {
	return Init(common_configs.GetConf().GameConf.ConfigPath)
}

// Init 从 JSON 配置目录加载全部配置表；path 为空则走 Go 内嵌占位。
//
// 加载失败返回 err 且不替换现有配置（保持旧配置）。
func Init(filePath string) error {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	if filePath == "" {
		return InitDefault()
	}
	gc, err := loadAll(filePath)
	if err != nil {
		loggers.Logger.Error("game_conf init failed", zap.String("path", filePath), zap.Error(err))
		return err
	}
	defaultConf.Store(gc)
	return nil
}

// InitDefault 加载 Go 内嵌占位配置（不走 JSON、不校验），供测试/无配置路径环境。
func InitDefault() error {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	gc := newEmbedded()
	gc.globalVersion = nextVersion()
	defaultConf.Store(gc)
	return nil
}

// InitBattle 轻量初始化战斗配置子集（battle 节点专用）：
// config_path 非空则从 JSON 加载 battle+skill 子集，否则 Go 内嵌子集。
func InitBattle() error {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	path := common_configs.GetConf().GameConf.ConfigPath
	if path == "" {
		gc := newBattleEmbedded()
		gc.globalVersion = nextVersion()
		defaultConf.Store(gc)
		return nil
	}
	gc, err := loadTablesFrom(path, battleTables)
	if err != nil {
		loggers.Logger.Error("game_conf init battle failed", zap.String("path", path), zap.Error(err))
		return err
	}
	defaultConf.Store(gc)
	return nil
}

// New 兼容入口：path 为空返回内嵌配置，否则从 JSON 加载（不替换全局单例）。
func New(filePath string) (*GameConf, error) {
	if filePath == "" {
		return newEmbedded(), nil
	}
	return loadAll(filePath)
}

// Load 加载全局配置（永不返回 nil）。
func Load() *GameConf {
	return defaultConf.Load()
}

// ReLoad 按当前 filePath 重新加载配置。
//
//   - filePath 为空 → 跳过（返回 nil）
//   - 加载失败 → 记录日志、保持旧配置（返回 err）
//   - 各表内容 hash 全同 → 跳过原子替换（mtime 触碰但内容未变不误触发）
func ReLoad() error {
	reloadMu.Lock()
	defer reloadMu.Unlock()
	old := defaultConf.Load()
	if old == nil || old.filePath == "" {
		loggers.Logger.Warn("game_conf reload skipped: json path not set")
		return nil
	}

	var gc *GameConf
	var err error
	if old.battleOnly {
		gc, err = loadTablesFrom(old.filePath, battleTables)
	} else {
		gc, err = loadAll(old.filePath)
	}
	if err != nil {
		loggers.Logger.Error("game_conf reload failed, keep old", zap.String("path", old.filePath), zap.Error(err))
		return err
	}
	if versionsEqual(old.tableVersions, gc.tableVersions) {
		loggers.Logger.Info("game_conf reload: content unchanged, skip swap")
		return nil
	}
	defaultConf.Store(gc)
	loggers.Logger.Info("game_conf reload success", zap.Uint64("version", gc.globalVersion))
	return nil
}

// nextVersion 全局版本递增（基于当前单例；无单例时从 1 起）。
func nextVersion() uint64 {
	if old := defaultConf.Load(); old != nil {
		return old.globalVersion + 1
	}
	return 1
}
