package login_game_clients

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_common"
)

// CreateRole 调对应 game 节点创建角色（出生点/主城/游戏数据落库由 game 完成）
func (c *GameClient) CreateRole(ctx context.Context, serverID uint32, req *pb_common.CreateRoleReq) (*pb_common.CreateRoleRsp, error) {
	cli := c.hub(serverID).GetGameHandlerClient()
	if cli == nil {
		return nil, fmt.Errorf("game[%d] not connected", serverID)
	}
	return cli.CreateRole(ctx, req)
}
