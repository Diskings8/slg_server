package worldmap_client

import (
	"context"
	"fmt"
	"sync"

	"server.slg.com/api/protocol/pb/pb_worldmap"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
)

// Client worldmap gRPC 客户端门面
//
// 基础连接维护在 rpcconn（rpc_conns 连接池 + rpc_handlers 生成的 hub，按 instance 感知发现），
// 这里只持有 hub 并提供 worldmap 的业务方法（Unary）与视野流（Stream）管理。
type Client struct {
	hub        *rpc_handlers.ClientHandler
	streamLock sync.RWMutex
	streams    map[uint64]*RoleStream
}

// New 创建 worldmap 客户端门面，hub 由 game_rpc_clients 初始化（绑定 game 实例）
func New(hub *rpc_handlers.ClientHandler) *Client {
	return &Client{
		hub:     hub,
		streams: make(map[uint64]*RoleStream),
	}
}

// NewStream 创建到 worldmap 的双向流（玩家视野流）
func (c *Client) NewStream(ctx context.Context) (pb_worldmap.WorldMapService_StreamClient, error) {
	svc := c.hub.GetWorldMapServiceClient()
	if svc == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return svc.Stream(ctx)
}

// Close 关闭底层 gRPC 连接（进程退出时调用）
func (c *Client) Close() {
	if c.hub != nil {
		c.hub.Close()
	}
}

// ------------------------------- 业务方法 -------------------------------

// CreateMarch 创建行军
func (c *Client) CreateMarch(ctx context.Context, req *pb_worldmap.CreateMarchReq) (*pb_worldmap.CreateMarchRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CreateMarch(ctx, req)
}

// SpawnReviewEvent 审查任务刷事件（worldmap 在主城 5×5 外圈生成 OverlayEvent）
func (c *Client) SpawnReviewEvent(ctx context.Context, req *pb_worldmap.SpawnReviewEventReq) (*pb_worldmap.SpawnReviewEventRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.SpawnReviewEvent(ctx, req)
}

// EventClick 气泡点击事件（采集/寻宝）：+进度，超 100% 完成（worldmap 返回事件类型供发奖）
func (c *Client) EventClick(ctx context.Context, req *pb_worldmap.EventClickReq) (*pb_worldmap.EventClickRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.EventClick(ctx, req)
}

// CreateRole 创建角色主城（分配出生点并落主城）
func (c *Client) CreateRole(ctx context.Context, req *pb_worldmap.CreateRoleReq) (*pb_worldmap.CreateRoleRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CreateRole(ctx, req)
}

// CancelMarch 取消行军
func (c *Client) CancelMarch(ctx context.Context, req *pb_worldmap.CancelMarchReq) (*pb_worldmap.CancelMarchRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.CancelMarch(ctx, req)
}

// MarchInfo 查询行军信息
func (c *Client) MarchInfo(ctx context.Context, req *pb_worldmap.MarchInfoReq) (*pb_worldmap.MarchInfoRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MarchInfo(ctx, req)
}

// MapData 查询地图数据
func (c *Client) MapData(ctx context.Context, req *pb_worldmap.MapDataReq) (*pb_worldmap.MapDataRsp, error) {
	cli := c.hub.GetWorldMapHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("worldmap not connected")
	}
	return cli.MapData(ctx, req)
}
