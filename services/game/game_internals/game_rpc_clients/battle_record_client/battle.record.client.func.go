package battle_record_client

// battle_record 客户端门面 — 玩家战报查询。

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/common/conns/rpcconn/rpc_handlers"
)

// Client battle_record gRPC 客户端门面
//
// 底层连接维护在 rpcconn（rpc_conns 连接池 + rpc_handlers 生成的 hub，按 instance 感知发现）。
type Client struct {
	hub *rpc_handlers.ClientHandler
}

// New 创建 battle_record 客户端门面，hub 由 game_rpc_clients 初始化（绑定 game 实例）
func New(hub *rpc_handlers.ClientHandler) *Client {
	return &Client{hub: hub}
}

// Close 关闭底层 gRPC 连接（进程退出时调用）
func (c *Client) Close() {
	if c.hub != nil {
		c.hub.Close()
	}
}

// ListBattleRecords 按 tag（角色/联盟/地块）分页查询战报
func (c *Client) ListBattleRecords(ctx context.Context, req *pb_battle_record.ListBattleRecordsReq) (*pb_battle_record.ListBattleRecordsRsp, error) {
	cli := c.hub.GetBattleRecordHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("battle_record not connected")
	}
	return cli.ListBattleRecords(ctx, req)
}

// ListBattleRecordChildren 查询主战报的子战报（车轮战 n 队整合）
func (c *Client) ListBattleRecordChildren(ctx context.Context, req *pb_battle_record.ListBattleRecordChildrenReq) (*pb_battle_record.ListBattleRecordChildrenRsp, error) {
	cli := c.hub.GetBattleRecordHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("battle_record not connected")
	}
	return cli.ListBattleRecordChildren(ctx, req)
}
