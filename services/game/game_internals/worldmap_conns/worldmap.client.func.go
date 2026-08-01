package worldmap_conns

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

// worldMapClient worldmap 客户端连接管理
//
// 全局连接状态收拢于此，通过 Wm() 获取单例。
type worldMapClient struct {
	cli    pb_worldmap.WorldMapHandlerClient
	conn   *grpc.ClientConn
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// wmClient 全局唯一 worldmap 客户端实例
var wmClient = &worldMapClient{}

// Wm 获取 worldmap 客户端单例
func Wm() *worldMapClient {
	return wmClient
}

// Init 初始化 worldmap 客户端，通过 ETCD 发现 worldmap 地址
// 启动后台 goroutine 定期刷新连接
func (c *worldMapClient) Init(parentCtx context.Context) {
	c.ctx, c.cancel = context.WithCancel(parentCtx)
	go c.watchConnect()
}

// watchConnect 定期检测 worldmap 连接，断开自动重连
func (c *worldMapClient) watchConnect() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 首次连接
	c.connect()

	for {
		select {
		case <-c.ctx.Done():
			c.closeConn()
			return
		case <-ticker.C:
			c.connect()
		}
	}
}

// connect 连接 worldmap
func (c *worldMapClient) connect() {
	addr, err := etcdconn.GetNodeTypeServerAddr(common_declarations.NodeWorldMapService)
	if err != nil {
		loggers.Logger.Warn("worldmap not found in etcd", zap.Error(err))
		return
	}

	c.mu.RLock()
	if c.conn != nil {
		// 已连接且目标没变就不重连
		if c.conn.Target() == addr {
			c.mu.RUnlock()
			return
		}
	}
	c.mu.RUnlock()

	// 关闭旧连接
	c.closeConn()

	// 建新连接
	dialCtx, dialCancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer dialCancel()

	newConn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		loggers.Logger.Warn("connect worldmap failed", zap.String("addr", addr), zap.Error(err))
		return
	}

	c.mu.Lock()
	c.conn = newConn
	c.cli = pb_worldmap.NewWorldMapHandlerClient(newConn)
	c.mu.Unlock()

	loggers.Logger.Info("worldmap client connected", zap.String("addr", addr))
}

// closeConn 关闭连接
func (c *worldMapClient) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.cli = nil
	}
}

// getClient 获取客户端（线程安全）
func (c *worldMapClient) getClient() pb_worldmap.WorldMapHandlerClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cli
}

// getConn 获取已建立的 gRPC 连接（线程安全）
func (c *worldMapClient) getConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

// NewStream 创建到 worldmap 的双向流（玩家视野流）
func (c *worldMapClient) NewStream(ctx context.Context) (pb_worldmap.WorldMapService_StreamClient, error) {
	conn := c.getConn()
	if conn == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return pb_worldmap.NewWorldMapServiceClient(conn).Stream(ctx)
}

// ------------------------------- 业务方法 -------------------------------

// CreateMarch 创建行军
func (c *worldMapClient) CreateMarch(ctx context.Context, req *pb_worldmap.CreateMarchReq) (*pb_worldmap.CreateMarchRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CreateMarch(ctx, req)
}

// CancelMarch 取消行军
func (c *worldMapClient) CancelMarch(ctx context.Context, req *pb_worldmap.CancelMarchReq) (*pb_worldmap.CancelMarchRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CancelMarch(ctx, req)
}

// MarchInfo 查询行军信息
func (c *worldMapClient) MarchInfo(ctx context.Context, req *pb_worldmap.MarchInfoReq) (*pb_worldmap.MarchInfoRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MarchInfo(ctx, req)
}

// MapData 查询地图数据
func (c *worldMapClient) MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MapData(ctx, req)
}
