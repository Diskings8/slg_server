package main

// battle_record
//
// 战斗记录节点：持久化战报（纯 MySQL，主表 + 标签索引表），提供 Save/Get/List RPC，
// 支持角色/联盟/地块三维查询，战报保留 14 天。
// worldmap 战斗结算后调用 SaveBattleRecord，game 玩家查询时调用 ListBattleRecords。

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
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/configs"
	"server.slg.com/common/conns/dbconn"
	"server.slg.com/common/conns/etcdconn"
	vgc "server.slg.com/common/globals/common_globals"
	"server.slg.com/common/loggers"
	"server.slg.com/common/servers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/battle_record/battle_handlers/battle_servers"
	"server.slg.com/services/battle_record/battle_internals/battle_records"
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

	// 节点地址优先取命令行，否则从配置读取（slg.dev.yaml 的 battle_record.addr）
	if rpcAddr == "" {
		rpcAddr = common_configs.GetConf().BattleRecord.Dsn()
	}

	loggers.Logger.Info("battle_record 服务启动",
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
		pb_battle_record.RegisterBattleRecordHandlerServer(grpcServer, battle_servers.BattleRecordServerHandler)
	}

	loggers.Logger.Info("gRPC 服务注册完成", zap.String("addr", rpcAddr))

	lis, err := net.Listen("tcp", rpcAddr)
	if err != nil {
		loggers.Logger.Fatal("gRPC 监听失败", zap.Error(err))
	}

	servers.NewLifecycle(
		servers.WithAsyncInit(
			func() {
				// 雪花 ID（战报主键由应用层生成）
				snowflakes.Init()

				// DB 初始化 + 建表 + 注入存储 + 启动清理
				dbconn.MustInitDB("mysql",
					common_configs.GetConf().DB.Game.DsnWithInstance(*vgc.CommonGlobalVarInstance),
					common_configs.GetConf().DB.Game.DsnWithInstance(*vgc.CommonGlobalVarInstance))

				store := battle_records.New(dbconn.GormDB())
				if err := store.Migrate(); err != nil {
					loggers.Logger.Fatal("战报表结构初始化失败", zap.Error(err))
				}
				battle_servers.BattleRecordServerHandler.SetStore(store)

				// 14 天过期清理（监听全局 ctx 退出）
				go cleanupLoop(ctx, store)

				// 必须初始化 etcd client，否则 SyncInit 注册时 etcdClient 为 nil
				etcdconn.InitEtcd(common_configs.GetConf().Etcd.Dsn())
				loggers.Logger.Info("DB/ETCD 初始化完成")
			},
		),

		servers.WithSyncInit(
			func() {
				etcdconn.RegisterServiceByNodeType(ctx, common_declarations.NodeBattleRecordService,
					*vgc.CommonGlobalVarInstance, rpcAddr)
				loggers.Logger.Info("ETCD 服务注册完成", zap.String("addr", rpcAddr))
			},
		),

		servers.WithShutdown(
			func() {
				loggers.Logger.Info("battle_record 关闭...")
			},
		),

		servers.WithGrpcServer(grpcServer, lis),
	).Run(ctx)

	loggers.Logger.Info(fmt.Sprintf("battle_record 服务已完全关闭 [%s]", *vgc.CommonGlobalVarInstance))
}

// cleanupLoop 每小时清理 14 天前的过期战报
func cleanupLoop(ctx context.Context, store *battle_records.Store) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-battle_records.RetentionDays * 24 * time.Hour).Unix()
			if err := store.CleanupExpired(cutoff); err != nil {
				loggers.Logger.Warn("清理过期战报失败", zap.Error(err))
			} else {
				loggers.Logger.Info("过期战报清理完成")
			}
		}
	}
}
