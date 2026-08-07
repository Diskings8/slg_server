package login_logics

// 区服列表业务逻辑。

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_account"
	"server.slg.com/services/login/login_internals/login_servers_store"
)

// ServerList 区服列表（客户端选服）
func ServerList(ctx context.Context, req *pb_account.ServerListReq) (*pb_account.ServerListResp, error) {
	if err := requireStore(); err != nil {
		return nil, err
	}

	servers, err := login_servers_store.Get().ListServers()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list servers failed: %v", err)
	}

	resp := &pb_account.ServerListResp{}
	for _, sv := range servers {
		resp.Servers = append(resp.Servers, &pb_account.ServerSimpleInfo{
			ServerId:   sv.ID,
			ServerName: sv.ServerName,
			Status:     sv.Status,
			OpenTime:   sv.OpenTime,
		})
	}
	return resp, nil
}
