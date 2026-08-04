package main

// game
//
// 业务逻辑，每一个 game 表示一个区服

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_game"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/services/game/game_handlers/game_servers"
	"server.slg.com/services/game/game_handlers/game_streams"
	"server.slg.com/services/game/game_internals"
)

var cfgFormat string

func parseFlagVar() {
	flag.StringVar(vgc.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
	flag.StringVar(vgc.CommonGlobalVarInstance, "instance", "0", "运行实例id")
	flag.StringVar(&cfgFormat, "config-format", "yaml", "配置格式: yaml / toml")
}

func main() {
	// 0. 参数解析
	parseFlagVar()
	flag.Parse()

	// 1. 配置 & 日志
	common_configs.LoadByFormat(cfgFormat, vgc.GetEnvPath())
	loggers.Init()
	loggers.Logger.Info("game 服务启动",
		zap.String("env", *vgc.CommonGlobalVarEnv),
		zap.String("instance", *vgc.CommonGlobalVarInstance))

	// 全局 context — 通过 cancel() 统一触发所有后台协程退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 信号监听 — 收到中断信号时取消 context
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		loggers.Logger.Info("收到关闭信号，开始优雅关闭...", zap.String("signal", sig.String()))
		cancel()
	}()

	// 构建 gRPC Server
	rpcAddr := common_configs.GetConf().Game.Dsn()
	grpcOpts := []grpc.ServerOption{
		grpc.ConnectionTimeout(5 * time.Second),
		grpc.MaxRecvMsgSize(10 * 1024 * 1024),
		grpc.MaxSendMsgSize(10 * 1024 * 1024),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:                  10 * time.Second,
			Timeout:               3 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	// 注册 gRPC 服务
	grpcServer := grpc.NewServer(grpcOpts...)
	{
		pb_game.RegisterGameServiceServer(grpcServer, game_streams.GameStreamHandler)
		pb_game.RegisterGameHandlerServer(grpcServer, game_servers.GameServerHandler)
	}

	loggers.Logger.Info("gRPC 服务注册完成", zap.String("addr", rpcAddr))

	// 创建 TCP 监听器
	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}

	// 启动生命周期 — 异步初始化 → 同步初始化 → gRPC/TCP 服务 → 等待关闭 → 清理退出
	servers.NewLifecycle(
		servers.WithAsyncInit(
			func() {
				dbconn.MustInitDB("mysql",
					common_configs.GetConf().DB.Game.Dsn(),
					common_configs.GetConf().DB.Game.Dsn())
				loggers.Logger.Info("数据库初始化完成")
			},
			func() {
				if err := game_conf.InitDefault(); err != nil {
					loggers.Logger.Error("game config init failed", zap.Error(err))
				}
				game_internals.Init(ctx)
				etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())
				loggers.Logger.Info("ETCD 初始化完成")
			},
		),

		servers.WithSyncInit(
			func() {
				etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeGameService,
					*vgc.CommonGlobalVarInstance, rpcAddr)
				loggers.Logger.Info("ETCD 服务注册完成", zap.String("addr", rpcAddr))
			},
		),

		servers.WithShutdown(
			func() {
				game_internals.ShutDown()
				loggers.Logger.Info("清理资源...")
			},
		),

		servers.WithGrpcServer(grpcServer, lis),
	).Run(ctx)

	loggers.Logger.Info(fmt.Sprintf("game 服务已完全关闭 [%s]", *vgc.CommonGlobalVarInstance))
}
