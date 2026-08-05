package main

// battle
//
// 战斗结算节点：提供 BattleSettle RPC，接收 worldmap 行军到达(attack_march)时的战斗计算请求。
// 纯计算服务——不持有 DB / 地图引擎，结算逻辑在 battle_internals/battle_logics。
// 与 game/worldmap 按 instance 单例配对，通过 etcd 注册发现。

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
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/services/battle/battle_handlers/battle_servers"
)

var (
	rpcAddr   string
	cfgFormat string
)

func parseFlagVar() {
	flag.StringVar(vgc.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
	flag.StringVar(vgc.CommonGlobalVarInstance, "instance", "0", "运行实例id")
	flag.StringVar(&rpcAddr, "addr", "", "监听地址，留空则从配置读取")
	flag.StringVar(&cfgFormat, "config-format", "yaml", "配置格式: yaml / toml")
}

func main() {
	parseFlagVar()
	flag.Parse()

	// 通用配置（DB/Redis/Etcd）统一加载
	common_configs.LoadByFormat(cfgFormat, vgc.GetEnvPath())
	loggers.Init()

	// 战斗配置（战斗规则 + 技能表）加载，供战斗结算使用；不加载英雄属性表/道具等通用配置
	if err := game_conf.InitBattle(); err != nil {
		loggers.Logger.Error("battle config init failed", zap.Error(err))
	}

	// 节点地址优先取命令行，否则从配置读取（slg.dev.yaml 的 battle.addr）
	if rpcAddr == "" {
		rpcAddr = common_configs.GetConf().Battle.Dsn()
	}

	loggers.Logger.Info("battle 服务启动",
		zap.String("env", *vgc.CommonGlobalVarEnv),
		zap.String("instance", *vgc.CommonGlobalVarInstance),
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

	// 注册 gRPC 服务
	{
		pb_battle.RegisterBattleHandlerServer(grpcServer, battle_servers.BattleServerHandler)
	}

	loggers.Logger.Info("gRPC 服务注册完成", zap.String("addr", rpcAddr))

	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}

	servers.NewLifecycle(
		servers.WithAsyncInit(
			func() {
				// 必须初始化 etcd client，否则 SyncInit 注册时 etcdClient 为 nil
				etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())
				loggers.Logger.Info("ETCD 初始化完成")
			},
		),

		servers.WithSyncInit(
			func() {
				etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeBattleService,
					*vgc.CommonGlobalVarInstance, rpcAddr)
				loggers.Logger.Info("ETCD 服务注册完成", zap.String("addr", rpcAddr))
			},
		),

		servers.WithShutdown(
			func() {
				loggers.Logger.Info("battle 关闭...")
			},
		),

		servers.WithGrpcServer(grpcServer, lis),
	).Run(ctx)

	loggers.Logger.Info(fmt.Sprintf("battle 服务已完全关闭 [%s]", *vgc.CommonGlobalVarInstance))
}
