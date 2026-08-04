package battle_record_handler

import (
	"context"

	"google.golang.org/protobuf/proto"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/common/conns/rpcconn/rpc_results"
	"server.slg.com/services/game/game_internals/game_rpc_clients"
)

// HandlerBattleRecordList 查询战报列表 (1000013)
//
// 查询维度由 tag_type 指定，转发到 battle_record 节点：
//   - 默认（未指定）：查本角色战报（ROLE，强制绑定本人 roleID，防越权）
//   - TAG_TYPE_ROLE：强制查本人
//   - TAG_TYPE_UNION / TAG_TYPE_TILE：透传客户端 tag_id（联盟/地块战报）
func HandlerBattleRecordList(ctx context.Context, roleID uint64,
	req *pb_battle_record.ListBattleRecordsReq, resp *pb_battle_record.ListBattleRecordsRsp) rpc_results.ResultI {

	// 默认查本人角色战报
	if req.GetTagType() == pb_battle_record.TagType_TAG_TYPE_UNKNOWN {
		req.TagType = pb_battle_record.TagType_TAG_TYPE_ROLE
	}
	// 角色战报强制绑定本人（防越权查询他人战报）
	if req.GetTagType() == pb_battle_record.TagType_TAG_TYPE_ROLE {
		req.TagId = roleID
	}
	if req.GetTagId() == 0 {
		return rpc_results.Error(pb_error_code.ErrorCode_ParamError, "invalid tag_id")
	}

	rsp, err := game_rpc_clients.BattleRecord().ListBattleRecords(ctx, req)
	if err != nil {
		return rpc_results.Error(pb_error_code.ErrorCode_Failed, err.Error())
	}
	proto.Merge(resp, rsp) // 合并到预创建响应（不复制 proto 内部锁状态）
	return nil
}
