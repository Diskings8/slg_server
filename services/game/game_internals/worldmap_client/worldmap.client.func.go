package worldmap_client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/conns/etcdconn"
	"server.slg.com/common/loggers"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	cli    pb_worldmap.WorldMapHandlerClient
	conn   *grpc.ClientConn
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
)

// Init 初始化 worldmap 客户端，通过 ETCD 发现 worldmap 地址
// 启动后台 goroutine 定期刷新连接
func Init(parentCtx context.Context) {
	ctx, cancel = context.WithCancel(parentCtx)
	go watchConnect()
}

// watchConnect 定期检测 worldmap 连接，断开自动重连
func watchConnect() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 首次连接
	connect()

	for {
		select {
		case <-ctx.Done():
			closeConn()
			return
		case <-ticker.C:
			connect()
		}
	}
}

// connect 连接 worldmap
func connect() {
	addr, err := etcdconn.GetNodeTypeServerAddr(common_declarations.NodeWorldMapService)
	if err != nil {
		loggers.Logger.Warn("worldmap not found in etcd", zap.Error(err))
		return
	}

	mu.RLock()
	if conn != nil {
		// 已连接且目标没变就不重连
		if conn.Target() == addr {
			mu.RUnlock()
			return
		}
	}
	mu.RUnlock()

	// 关闭旧连接
	closeConn()

	// 建新连接
	dialCtx, dialCancel := context.WithTimeout(ctx, 5*time.Second)
	defer dialCancel()

	newConn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		loggers.Logger.Warn("connect worldmap failed", zap.String("addr", addr), zap.Error(err))
		return
	}

	mu.Lock()
	conn = newConn
	cli = pb_worldmap.NewWorldMapHandlerClient(newConn)
	mu.Unlock()

	loggers.Logger.Info("worldmap client connected", zap.String("addr", addr))
}

func closeConn() {
	mu.Lock()
	defer mu.Unlock()
	if conn != nil {
		conn.Close()
		conn = nil
		cli = nil
	}
}

// getClient 获取客户端（线程安全）
func getClient() pb_worldmap.WorldMapHandlerClient {
	mu.RLock()
	defer mu.RUnlock()
	return cli
}

// ------------------------------- 业务方法 -------------------------------

// CreateMarch 创建行军
func CreateMarch(ctx context.Context, req *pb_worldmap.CreateMarchReq) (*pb_worldmap.CreateMarchRsp, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return c.CreateMarch(ctx, req)
}

// CancelMarch 取消行军
func CancelMarch(ctx context.Context, req *pb_worldmap.CancelMarchReq) (*pb_worldmap.CancelMarchRsp, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return c.CancelMarch(ctx, req)
}

// MarchInfo 查询行军信息
func MarchInfo(ctx context.Context, req *pb_worldmap.MarchInfoReq) (*pb_worldmap.MarchInfoRsp, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return c.MarchInfo(ctx, req)
}

// MapData 查询地图数据
func MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	c := getClient()
	if c == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return c.MapData(ctx, req)
}
