package game_conf

import (
	"sync/atomic"

	"server.slg.com/api/game_conf/battle"
	"server.slg.com/api/game_conf/gacha"
	"server.slg.com/api/game_conf/hero"
	"server.slg.com/api/game_conf/item"
	"server.slg.com/api/game_conf/skill"
	"server.slg.com/api/game_conf/troop"
	"server.slg.com/api/protocol/pb_confs"
	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/loggers"
)

var defaultConf atomic.Pointer[GameConf]

// init 兜底：确保默认配置可用（Load 永不返回 nil），供 battle 等纯计算服务使用
func init() {
	_ = InitDefault()
}

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
		Battle:  battle.New(),
		Hero:    hero.New(),
		Skill:   skill.New(),
		Item:    item.New(),
		Troop:   troop.New(),
		Gacha:   gacha.New(),
	}
	defaultConf.Store(gc)
	return nil
}

// InitBattle 轻量初始化战斗配置子集（battle 节点专用）：
// 只加载战斗规则 + 技能表，不加载英雄属性表/道具/建筑等通用配置。
func InitBattle() error {
	gc := &GameConf{
		configs: &pb_confs.Table{},
		Battle:  battle.New(),
		Skill:   skill.New(),
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
		Gacha:    gacha.New(),
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
