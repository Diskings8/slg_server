package game_conf

import (
	"sync/atomic"

	"server.slg.com/api/protocol/pb_confs"
	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/loggers"
)

var defaultConf atomic.Pointer[GameConf]

// InitFromConf 使用EnvConf配置初始化配置
func InitFromConf() error {
	filePath := common_configs.GetConf().GameConf.ConfigPath
	return Init(filePath)
}

// Init 根据配置路径初始化配置内容
func Init(filePath string) error {
	ccc, err := New(filePath)
	if err == nil {
		defaultConf.Store(ccc)
	}
	return err
}

// New 新配置路径加载配置
func New(filePath string) (*GameConf, error) {
	gameConfig := &GameConf{
		configs:  &pb_confs.Table{},
		filePath: filePath,
	}
	if err := gameConfig.init(); err != nil {
		return nil, err
	}
	return gameConfig, nil
}

// Load 加载游戏配置
func Load() *GameConf {
	return defaultConf.Load()
}

// ReLoad 重新加载配置
func ReLoad() {
	newConf := new(GameConf)
	err := newConf.init()
	if err != nil {
		loggers.Logger.Error(err.Error())
	} else {
		defaultConf.Swap(newConf)
	}
}
