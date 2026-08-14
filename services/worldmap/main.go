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
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/cacheconn"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/common/utils/crontabs"
	"server.slg.com/common/utils/snowflakes"
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

	// 节点地址优先取命令行，否则从全局配置读取（worldmap.addr）
	if rpcAddr == "" {
		rpcAddr = common_configs.GetConf().Worldmap.Addr
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
	worldmap_streams.WorldMapStreamHandler.SetEngine(engine)
	worldmap_servers.WorldMapServerHandler.SetEngine(engine)

	// 注册 gRPC 服务
	{
		pb_worldmap.RegisterWorldMapServiceServer(grpcServer, worldmap_streams.WorldMapStreamHandler)
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
				// 雪花 ID（行军主键由应用层生成，与 game/login 保持一致；须在 DB 初始化前）
				snowflakes.Init()

				dbconn.MustInitDB("mysql",
					common_configs.GetConf().DB.Worldmap.DsnWithInstance(*vgc.CommonGlobalVarInstance),
					common_configs.GetConf().DB.Worldmap.DsnWithInstance(*vgc.CommonGlobalVarInstance))
				loggers.Logger.Info("数据库初始化完成")

				// etcd 客户端：SyncInit 里 RegisterServiceByNodeType 依赖它，缺了会 nil panic（对齐 game/battle/battle_record）
				etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())

				// redis：roles poller 定时保存依赖 cacheconn（SRandMember 刷盘队列）；缺了 cron 触发时 nil panic
				if err := cacheconn.Init(ctx); err != nil {
					loggers.Logger.Fatal("redis 初始化失败", zap.Error(err))
				}

				// 游戏配置：守军配置（开发行军战斗对象）；config_path 非空走 JSON，为空走 Go 内嵌占位
				if err := game_conf.InitFromConf(); err != nil {
					loggers.Logger.Error("game config init failed", zap.Error(err))
				}

				// 角色数据 poller：主城落位回写 role_data（须在 DB 初始化后调用）
				roles.Init(ctx)
				loggers.Logger.Info("roles poller 初始化完成")

				// 全局定时器：驱动异步保存（行军/地块定时刷盘）；此前全仓库无调用点，
				// SaveDo 从未真正执行，行军/地块只是停留在内存等待队列
				crontabs.Start()

				// 地图恢复：DB 动态状态覆盖种子生成的底图（稀疏覆盖模型）
				if err := engine.InitMapData(); err != nil {
					loggers.Logger.Error("map data load failed", zap.Error(err))
				}

				// 行军恢复：从 DB 加载 + 重做 AOI 注册
				if err := engine.InitMarchs(); err != nil {
					loggers.Logger.Error("march recovery failed", zap.Error(err))
				}
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
				// 停止全局定时器（阻塞直到 cron 退出），随后 gRPC 关闭
				crontabs.ShutDown()
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
