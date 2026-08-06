package main

// login
//
// 账号登录节点（跨服全局单例，instance=0）：注册/登录账号、账号下角色列表、区服列表、
// 进入区服（建角调 game.CreateRole）。账号/角色映射/区服数据落 common_db_0。
// 客户端经 gateway 接入（网关路由为后续工作），服务间经 gRPC + etcd 发现。

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
	"google.golang.org/grpc/reflection"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/login/login_handlers/login_servers"
	"server.slg.com/services/login/login_internals"
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

	// 节点地址优先取命令行，否则从配置读取（slg.dev.yaml 的 login.addr）
	if rpcAddr == "" {
		rpcAddr = common_configs.GetConf().Login.Dsn()
	}

	loggers.Logger.Info("login 服务启动",
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

	// 注册 gRPC 服务（+ reflection 便于 grpcurl 调试）
	{
		pb_account.RegisterAccountServiceServer(grpcServer, login_servers.LoginServerHandler)
		reflection.Register(grpcServer)
	}

	loggers.Logger.Info("gRPC 服务注册完成", zap.String("addr", rpcAddr))

	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}

	servers.NewLifecycle(
		servers.WithAsyncInit(
			func() {
				// 雪花 ID（账号/角色主键由应用层生成）
				snowflakes.Init()

				// DB 初始化（账号/角色/区服数据在 common_db_0）+ 建表 + 种子区服
				dbconn.MustInitDB("mysql",
					common_configs.GetConf().DB.Common.Dsn(),
					common_configs.GetConf().DB.Common.Dsn())

				login_internals.Init()

				// 必须初始化 etcd client，否则 SyncInit 注册时 etcdClient 为 nil
				etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())
				loggers.Logger.Info("DB/ETCD 初始化完成")
			},
		),

		servers.WithSyncInit(
			func() {
				etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeLoginService,
					*vgc.CommonGlobalVarInstance, rpcAddr)
				loggers.Logger.Info("ETCD 服务注册完成", zap.String("addr", rpcAddr))
			},
		),

		servers.WithShutdown(
			func() {
				login_internals.Shutdown()
				loggers.Logger.Info("login 关闭...")
			},
		),

		servers.WithGrpcServer(grpcServer, lis),
	).Run(ctx)

	loggers.Logger.Info(fmt.Sprintf("login 服务已完全关闭 [%s]", *vgc.CommonGlobalVarInstance))
}
