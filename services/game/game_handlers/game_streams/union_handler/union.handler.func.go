package union_handler

import (
	"context"
	"fmt"

	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_union"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/game/game_logics"
)

// HandlerUnionCreate 创建联盟 (1000044)
func HandlerUnionCreate(ctx context.Context, roleID uint64, req *pb_union.UnionCreateReq, resp *pb_union.UnionCreateResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	ru, logicErr := game_logics.UnionCreate(role, req.GetName())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union create failed: %s", logicErr.Error()))
	}
	poller.Save()
	resp.RoleUnion = ru
	return nil
}

// HandlerUnionJoin 加入联盟 (1000045)
func HandlerUnionJoin(ctx context.Context, roleID uint64, req *pb_union.UnionJoinReq, resp *pb_union.UnionJoinResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	ru, logicErr := game_logics.UnionJoin(role, req.GetUnionId())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union join failed: %s", logicErr.Error()))
	}
	poller.Save()
	resp.RoleUnion = ru
	return nil
}

// HandlerUnionLeave 退出联盟 (1000046)
func HandlerUnionLeave(ctx context.Context, roleID uint64, req *pb_union.UnionLeaveReq, resp *pb_union.UnionLeaveResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if logicErr := game_logics.UnionLeave(role); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union leave failed: %s", logicErr.Error()))
	}
	poller.Save()
	return nil
}

// HandlerUnionKick 踢人 (1000047)
func HandlerUnionKick(ctx context.Context, roleID uint64, req *pb_union.UnionKickReq, resp *pb_union.UnionKickResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if logicErr := game_logics.UnionKick(role, req.GetRoleId()); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union kick failed: %s", logicErr.Error()))
	}
	poller.Save()
	return nil
}

// HandlerUnionTransfer 转让盟主 (1000048)
func HandlerUnionTransfer(ctx context.Context, roleID uint64, req *pb_union.UnionTransferReq, resp *pb_union.UnionTransferResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if logicErr := game_logics.UnionTransfer(role, req.GetRoleId()); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union transfer failed: %s", logicErr.Error()))
	}
	poller.Save()
	return nil
}

// HandlerUnionDissolve 解散联盟 (1000049)
func HandlerUnionDissolve(ctx context.Context, roleID uint64, req *pb_union.UnionDissolveReq, resp *pb_union.UnionDissolveResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	if logicErr := game_logics.UnionDissolve(role); logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union dissolve failed: %s", logicErr.Error()))
	}
	poller.Save()
	return nil
}

// HandlerUnionInfo 查询联盟信息 (1000050)
func HandlerUnionInfo(ctx context.Context, roleID uint64, req *pb_union.UnionInfoReq, resp *pb_union.UnionInfoResp) rpc_results.ResultI {
	info, logicErr := game_logics.UnionInfo(req.GetUnionId())
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union info failed: %s", logicErr.Error()))
	}
	resp.Union = info
	return nil
}

// HandlerUnionMemberList 查询联盟成员列表 (1000051)
func HandlerUnionMemberList(ctx context.Context, roleID uint64, req *pb_union.UnionMemberListReq, resp *pb_union.UnionMemberListResp) rpc_results.ResultI {
	poller, role, err := game_roles.GetRole(roleID)
	if err != nil {
		return err
	}
	defer poller.Release()

	members, logicErr := game_logics.UnionMemberList(role)
	if logicErr != nil {
		if r, ok := logicErr.(rpc_results.ResultI); ok {
			return r
		}
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, fmt.Sprintf("union member list failed: %s", logicErr.Error()))
	}
	resp.Members = members
	return nil
}
