package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"server.slg.com/common/configs"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/services/game/mix_server_games"
)

func parseFlagVar() {
	flag.StringVar(vgc.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
	flag.StringVar(vgc.CommonGlobalVarInstance, "instance", "0", "运行实例id")
}

func main() {
	parseFlagVar()
	flag.Parse()

	configs.LoadEnvConf(vgc.GetEnvPath())

	loggers.Init()
	loggers.Log.Info("游戏服务启动")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// etcd
	rpcAddr := configs.GEnvConf.GameServer.Dsn()
	etcdconn.RegisterServiceByNodeType(etcdconn.NodeGameService, *vgc.CommonGlobalVarInstance, rpcAddr)

	// init rpc server
	conf := servers.Config{
		Addr:             rpcAddr,
		Timeout:          5 * time.Second,
		MaxRecvMsgSize:   10 * 1024 * 1024,
		MaxSendMsgSize:   10 * 1024 * 1024,
		EnableReflection: true,
	}

	srv := servers.BuildRpcServer(ctx, conf)
	srv.RegisterServices(&mix_server_games.GameServer{})

	serverCount := atomic.Int32{}
	serverChan := make(chan struct{}, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverCount.Add(1)
	go func() {
		if srv.Run() != nil {
			serverChan <- struct{}{}
		}
	}()

	for {
		select {
		case <-quit:
			loggers.Log.Info("收到关闭信号，开始优雅关闭服务...")
			cancel()
			time.Sleep(3 * time.Second)
			return
		case <-serverChan:
			serverCount.Add(-1)
			return
		}
	}
}
