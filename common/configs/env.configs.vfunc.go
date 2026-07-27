package common_configs

import (
	"os"

	"gopkg.in/yaml.v3"
	"server.slg.com/common/configs/env_confs"
)

var defaultEnvConf env_confs.Config

func LoadEnvConf(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("load common_configs fail: " + err.Error())
	}

	err = yaml.Unmarshal(data, &defaultEnvConf)
	if err != nil {
		panic("parse common_configs fail: " + err.Error())
	}
}

func GetEnvConf() *env_confs.Config {
	return &defaultEnvConf
}
