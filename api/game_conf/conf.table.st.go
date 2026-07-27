package game_conf

import (
	"fmt"
	"os"
	"sync"

	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/utils/util_jsons"
)

type GameConf struct {
	mu       sync.RWMutex
	configs  *pb_confs.Table
	filePath string
}

// AllConfigs 全部配置
func (c *GameConf) AllConfigs() *pb_confs.Table {
	return c.configs
}

func (c *GameConf) init() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	v, err := os.ReadFile(c.filePath)
	if err != nil {
		dir, _ := os.Getwd()
		return fmt.Errorf("read file failed, dir: %s, err: %s", dir, err.Error())
	}

	err = util_jsons.Unmarshal(v, c.configs)
	if err != nil {
		return fmt.Errorf("unmarshal json file failed, filePath: %s, err: %s", c.filePath, err.Error())
	}

	c.initMaps()
	return nil
}

func (c *GameConf) initMaps() {
	// todo
}
