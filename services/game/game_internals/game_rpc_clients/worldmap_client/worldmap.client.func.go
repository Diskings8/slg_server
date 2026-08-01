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
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// Client worldmap gRPC 客户端，封装连接管理
// 懒连接：首次业务访问时才建立连接，之后由 game_rpc_clients 统一 ticker 维护
type Client struct {
	cli           pb_worldmap.WorldMapHandlerClient
	conn          *grpc.ClientConn
	mu            sync.RWMutex
	ctx           context.Context
	everConnected bool
}

// New 创建 worldmap 客户端，ctx 用于连接时的超时控制
func New(parentCtx context.Context) *Client {
	return &Client{ctx: parentCtx}
}

// Connect 连接 worldmap。已就绪直接返回；连接断开或目标地址变更则重连
func (c *Client) Connect() {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn != nil && conn.GetState() == connectivity.Ready {
		return
	}

	addr, err := etcdconn.GetNodeTypeServerAddr(common_declarations.NodeWorldMapService)
	if err != nil {
		loggers.Logger.Warn("worldmap not found in etcd", zap.Error(err))
		return
	}

	c.mu.RLock()
	conn = c.conn
	c.mu.RUnlock()
	if conn != nil && conn.GetState() == connectivity.Ready && conn.Target() == addr {
		return
	}

	// 关闭旧连接
	c.Close()

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
	c.everConnected = true
	c.mu.Unlock()

	loggers.Logger.Info("worldmap client connected", zap.String("addr", addr))
}

// Close 关闭连接
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.cli = nil
	}
}

// Active 是否曾建立过连接（懒连接客户端未被使用过则无需由 ticker 维护）
func (c *Client) Active() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.everConnected
}

// getClient 获取客户端（线程安全）
func (c *Client) getClient() pb_worldmap.WorldMapHandlerClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cli
}

// ------------------------------- 业务方法 -------------------------------

// CreateMarch 创建行军
func (c *Client) CreateMarch(ctx context.Context, req *pb_worldmap.CreateMarchReq) (*pb_worldmap.CreateMarchRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CreateMarch(ctx, req)
}

// CancelMarch 取消行军
func (c *Client) CancelMarch(ctx context.Context, req *pb_worldmap.CancelMarchReq) (*pb_worldmap.CancelMarchRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CancelMarch(ctx, req)
}

// MarchInfo 查询行军信息
func (c *Client) MarchInfo(ctx context.Context, req *pb_worldmap.MarchInfoReq) (*pb_worldmap.MarchInfoRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MarchInfo(ctx, req)
}

// MapData 查询地图数据
func (c *Client) MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	cli := c.getClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MapData(ctx, req)
}
