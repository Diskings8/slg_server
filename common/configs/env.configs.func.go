package configs

import (
	"os"

	"gopkg.in/yaml.v3"
)

func LoadEnvConf(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic("load configx fail: " + err.Error())
	}

	err = yaml.Unmarshal(data, &GEnvConf)
	if err != nil {
		panic("parse configx fail: " + err.Error())
	}
}
