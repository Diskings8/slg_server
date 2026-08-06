package main

// worldmap
//
// 地图引擎节点，持有 cores 引擎（AOI/行军/战斗），接收 game 的行军请求

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
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/services/internal/cores/roles"
	"server.slg.com/services/worldmap/worldmap_handlers/worldmap_servers"
	"server.slg.com/services/worldmap/worldmap_handlers/worldmap_streams"
	worldmap_inits "server.slg.com/services/worldmap/worldmap_internals/worldmap_inits"
)

var (
	nodeName   string
	rpcAddr    string
	cfgFormat  string
)

func parseFlagVar() {
	flag.StringVar(vgc.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
	flag.StringVar(vgc.CommonGlobalVarInstance, "instance", "0", "运行实例id")
	flag.StringVar(&nodeName, "node", "worldmap-1", "节点名称，如 worldmap-1")
	flag.StringVar(&rpcAddr, "addr", "", "监听地址，留空则从 TOML 配置读取")
	flag.StringVar(&cfgFormat, "config-format", "yaml", "配置格式: yaml / toml")
}

func main() {
	parseFlagVar()
	flag.Parse()

	// 通用配置（DB/Redis/Etcd）统一加载
	common_configs.LoadByFormat(cfgFormat, vgc.GetEnvPath())
	loggers.Init()

	// 节点地址优先取命令行，否则从 TOML 节点配置读取
	if rpcAddr == "" {
		rpcAddr = common_configs.LoadNodeConfig(nodeName, "../../config", "config.dev")
	}

	loggers.Logger.Info("worldmap 服务启动",
		zap.String("env", *vgc.CommonGlobalVarEnv),
		zap.String("node", nodeName),
		zap.String("addr", rpcAddr))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-quit
		loggers.Logger.Info("收到关闭信号，开始优雅关闭...", zap.String("signal", sig.String()))
		cancel()
	}()

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
	grpcServer := grpc.NewServer(grpcOpts...)

	// 初始化 cores 引擎
	engine := worldmap_inits.NewEngine(ctx)
	worldmap_handlers.WorldMapStreamHandler.SetEngine(engine)
	worldmap_servers.WorldMapServerHandler.SetEngine(engine)

	// 注册 gRPC 服务
	{
		pb_worldmap.RegisterWorldMapServiceServer(grpcServer, worldmap_handlers.WorldMapStreamHandler)
		pb_worldmap.RegisterWorldMapHandlerServer(grpcServer, worldmap_servers.WorldMapServerHandler)
	}

	loggers.Logger.Info("gRPC 服务注册完成", zap.String("addr", rpcAddr))

	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}

	var engineStopped bool
	servers.NewLifecycle(
		servers.WithAsyncInit(
			func() {
				dbconn.MustInitDB("mysql",
					common_configs.GetConf().DB.Game.Dsn(),
					common_configs.GetConf().DB.Game.Dsn())
				loggers.Logger.Info("数据库初始化完成")

				// 角色数据 poller：主城落位回写 role_data（须在 DB 初始化后调用）
				roles.Init(ctx)
				loggers.Logger.Info("roles poller 初始化完成")
			},
		),

		servers.WithSyncInit(
			func() {
				etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeWorldMapService,
					*vgc.CommonGlobalVarInstance, rpcAddr)
				loggers.Logger.Info("ETCD 服务注册完成", zap.String("addr", rpcAddr))
			},
		),

		servers.WithShutdown(
			func() {
				if !engineStopped {
					engine.Stop()
					engineStopped = true
				}
				loggers.Logger.Info("worldmap 关闭...")
			},
		),

		servers.WithGrpcServer(grpcServer, lis),
	).Run(ctx)

	if !engineStopped {
		engine.Stop()
	}
	loggers.Logger.Info(fmt.Sprintf("worldmap 服务已完全关闭 [%s]", *vgc.CommonGlobalVarInstance))
}
