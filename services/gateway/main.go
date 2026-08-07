package main

import (
	"context"
	"flag"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"server.slg.com/common/common_declarations"
	common_configs "server.slg.com/common/configs"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/conns/etcdconn"
	"server.slg.com/common/conns/netconn/tcp_conn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/services/gateway/mix_server_gateways"
	"server.slg.com/services/gateway/session_gateways"
)

var cfgFormat string

func parseFlagVar() {
	flag.StringVar(vgc.CommonGlobalVarEnv, "env", "dev", "运行环境：dev/pre/prod")
	flag.StringVar(vgc.CommonGlobalVarInstance, "instance", "0", "运行实例id")
	flag.StringVar(&cfgFormat, "config-format", "yaml", "配置格式: yaml / toml")
}

func main() {
	parseFlagVar()
	flag.Parse()

	common_configs.LoadByFormat(cfgFormat, vgc.GetEnvPath())

	loggers.Init()
	loggers.Logger.Info("网关启动")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// etcd
	etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())
	rpcAddr := common_configs.GetConf().Gateway.RpcDsn()
	tcpAddr := common_configs.GetConf().Gateway.TcpDsn()
	etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeGatewayService, *vgc.CommonGlobalVarInstance, rpcAddr)

	// redis：订阅进服广播（踢旧连接）+ 路由表查询
	if err := cacheconn.Init(ctx); err != nil {
		loggers.Logger.Fatal("redis 初始化失败", zap.Error(err))
	}

	// 本 gateway 节点标识（唯一：instance + 监听地址）+ 启动进服广播监听
	session_gateways.SetNodeID(*vgc.CommonGlobalVarInstance + ":" + rpcAddr)
	session_gateways.StartRoleEnterWatcher(ctx)

	// Build gRPC server
	grpcLis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}
	grpcSrv := servers.NewGrpcBuilder().
		WithOptions(
			grpc.ConnectionTimeout(5*time.Second),
			grpc.MaxRecvMsgSize(10*1024*1024),
			grpc.MaxSendMsgSize(10*1024*1024),
		).
		WithService(&mix_server_gateways.MixServer{}).
		Build()
	reflection.Register(grpcSrv)

	// Build TCP server
	tcpSrv := servers.NewTcpBuilder(servers.Config{Addr: tcpAddr}).
		WithHandler(func(conn net.Conn) {
			// 全局 ctx 传入会话：game 流等长驻资源从它派生，关停时统一传导
			session_gateways.NewSession(ctx, tcp_conn.NewNetConn(conn)).RunToReceiveFromConn()
		}).
		Build()

	// Lifecycle 统一管理 gRPC + TCP 的启动和关闭
	ls := servers.NewLifecycle(
		servers.WithGrpcServer(grpcSrv, grpcLis),
		servers.WithTcpServer(tcpSrv),
		servers.WithShutdown(func() {
			_ = cacheconn.ShutDown() // 关闭 redis 连接
		}),
	)

	// 系统信号 → ctx cancel → Lifecycle 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		loggers.Logger.Info("收到关闭信号，开始优雅关闭服务...")
		cancel()
	}()

	if err := ls.Run(ctx); err != nil {
		loggers.Logger.Fatal("服务异常退出", zap.Error(err))
	}
	loggers.Logger.Info("服务已完全关闭")
}
