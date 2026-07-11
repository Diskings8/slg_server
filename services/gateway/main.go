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
	"server.slg.com/services/gateway/mix_server_gateways"
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
	loggers.Logger.Info("网关启动")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// pprof

	//etcd
	etcdconn.InitEtcd(configs.GEnvConf.Etcd.Dsn())
	rpcAddr := configs.GEnvConf.GateWay.RpcDsn()
	tcpAddr := configs.GEnvConf.GateWay.TcpDsn()
	etcdconn.RegisterServiceByNodeType(ctx, etcdconn.NodeGatewayService, *vgc.CommonGlobalVarInstance, rpcAddr)

	// init system

	//
	serverCount := atomic.Int32{}
	serverChan := make(chan struct{}, 2)
	// 主进程阻塞监听系统退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// init rpc server
	grpcServerFunc := func() {
		serverCount.Add(1)
		conf := servers.Config{
			Addr:             rpcAddr,
			Timeout:          5 * time.Second,
			MaxRecvMsgSize:   10 * 1024 * 1024,
			MaxSendMsgSize:   10 * 1024 * 1024,
			EnableReflection: true,
		}

		srv := servers.BuildRpcServer(ctx, conf)
		srv.RegisterServices(&mix_server_gateways.MixServer{})

		if srv.Run() != nil {
			serverChan <- struct{}{}
		}
	}

	tcpServerFunc := func() {
		serverCount.Add(1)
		conf := servers.Config{
			Addr:             tcpAddr,
			Timeout:          5 * time.Second,
			MaxRecvMsgSize:   10 * 1024 * 1024,
			MaxSendMsgSize:   10 * 1024 * 1024,
			EnableReflection: true,
		}
		srv := servers.BuildTcpServer(ctx, conf)
		if srv.Run() != nil {
			serverChan <- struct{}{}
		}
	}

	grpcServerFunc()
	tcpServerFunc()

	for {
		select {
		case <-quit:
			loggers.Logger.Info("收到关闭信号，开始优雅关闭服务...")
			cancel()
			time.Sleep(time.Second * 1)
			return
		case <-serverChan:
			serverCount.Add(-1)
			if serverCount.Load() == 0 {
				return
			}
		}
	}
}
