package game_conf

import (
	"sync/atomic"

	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/troop"
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

// InitDefault 加载 Go 内嵌配置（不走 JSON），供测试/未配置 JSON 环境使用
func InitDefault() error {
	gc := &GameConf{
		configs: &pb_confs.Table{},
		Hero:    hero.New(),
		Skill:   skill.New(),
		Item:    item.New(),
		Troop:   troop.New(),
	}
	defaultConf.Store(gc)
	return nil
}

// New 新配置路径加载配置
func New(filePath string) (*GameConf, error) {
	gameConfig := &GameConf{
		configs:  &pb_confs.Table{},
		filePath: filePath,
		Hero:     hero.New(),
		Skill:    skill.New(),
		Item:     item.New(),
		Troop:    troop.New(),
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
