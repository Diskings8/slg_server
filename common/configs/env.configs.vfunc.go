package common_configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"server.slg.com/common/configs/env_confs"
)

var defaultConf env_confs.Config

// ================================================================
// 统一加载入口
// ================================================================

// LoadByFormat 根据格式名加载配置文件，YAML / TOML 均映射到同一 Config 结构体。
//
//	format: "yaml" / "toml"
//	path:   配置文件全路径
func LoadByFormat(format, path string) {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		loadYaml(path)
	case "toml":
		loadToml(path)
	default:
		panic(fmt.Sprintf("unsupported config format: %s (supported: yaml, toml)", format))
	}
}

// LoadConfig 根据文件后缀自动识别格式并加载。
func LoadConfig(path string) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		loadYaml(path)
	case ".toml":
		loadToml(path)
	default:
		panic(fmt.Sprintf("unsupported config format: %s", path))
	}
}

// GetConf 返回统一配置（不关心底层是 YAML 还是 TOML）。
func GetConf() *env_confs.Config {
	return &defaultConf
}

// ================================================================
// 格式加载器
// ================================================================

func loadYaml(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("load yaml config fail: " + err.Error())
	}
	if err = yaml.Unmarshal(data, &defaultConf); err != nil {
		panic("parse yaml config fail: " + err.Error())
	}
}

func loadToml(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("load toml config fail: " + err.Error())
	}
	if err = toml.Unmarshal(data, &defaultConf); err != nil {
		panic("parse toml config fail: " + err.Error())
	}
}

// LoadYamlConf 显式加载 YAML 配置（兼容旧调用方）。
func LoadYamlConf(path string) {
	loadYaml(path)
}

// ================================================================
// 节点配置
// ================================================================

// LoadNodeConfig 从 TOML / YAML 文件读取节点地址。
// 适用于 worldmap 这类在配置文件中使用命名节点段的场景。
//
//	nodeName:   节点段名，如 "worldmap-1"
//	configPath: 配置文件目录
//	configName: 文件名（不含后缀）
func LoadNodeConfig(nodeName, configPath, configName string) string {
	v := viper.New()
	v.SetConfigName(configName)
	v.AddConfigPath(configPath)
	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("load node config fail: %s", err.Error()))
	}
	sub := v.Sub(nodeName)
	if sub == nil {
		panic(fmt.Sprintf("node config [%s] not found", nodeName))
	}
	if addr := sub.GetString("address"); addr != "" {
		return addr
	}
	return sub.GetString("tcp_address")
}
