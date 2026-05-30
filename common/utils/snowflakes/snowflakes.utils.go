package snowflakes

import (
	"github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
	"server.slg.com/common/configs"
	"server.slg.com/common/loggers"
)

var node *snowflake.Node

func Init() {
	// 节点ID 1~1023，专服可以配置
	// 从 YAML 读取配置
	cfg := configs.GEnvConf.Snowflake

	// 设置雪花算法参数
	snowflake.NodeBits = 10
	snowflake.StepBits = 12

	// 创建节点：使用 datacenter + worker 组合
	nodeID := (cfg.DatacenterID << 5) | cfg.WorkerID

	n, err := snowflake.NewNode(nodeID)
	if err != nil {
		loggers.Log.Fatal("雪花算法初始化失败", zap.Error(err))
		panic(err)
	}
	node = n

	loggers.Log.Info("雪花算法初始化成功",
		zap.Int64("datacenter_id", cfg.DatacenterID),
		zap.Int64("worker_id", cfg.WorkerID),
		zap.Int64("node_id", nodeID),
	)
}

// GenUID 生成全局唯一UID
func GenUID() int64 {
	return node.Generate().Int64()
}

// GenUUID 生成全局唯一UUID
func GenUUID() uint64 {
	return uint64(GenUID())
}

func GenStringID() string {
	return node.Generate().String()
}
