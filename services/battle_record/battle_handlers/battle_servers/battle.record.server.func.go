package battle_servers

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/services/battle_record/battle_internals/battle_records"
)

// SaveBattleRecord 保存战报（worldmap 战斗结算后调用）
func (s *BattleRecordServer) SaveBattleRecord(ctx context.Context, req *pb_battle_record.SaveBattleRecordReq) (*pb_battle_record.SaveBattleRecordRsp, error) {
	if s.store == nil {
		return nil, status.Error(codes.Internal, "store not initialized")
	}
	if req == nil || req.GetAttackerRoleId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "attacker_role_id required")
	}

	rec, err := battle_records.RecordFromReq(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid results: "+err.Error())
	}
	if err := s.store.SaveRecord(rec); err != nil {
		return nil, status.Error(codes.Internal, "save record failed: "+err.Error())
	}
	return &pb_battle_record.SaveBattleRecordRsp{RecordId: rec.ID}, nil
}

// GetBattleRecord 按战报 ID 查询
func (s *BattleRecordServer) GetBattleRecord(ctx context.Context, req *pb_battle_record.GetBattleRecordReq) (*pb_battle_record.GetBattleRecordRsp, error) {
	if s.store == nil {
		return nil, status.Error(codes.Internal, "store not initialized")
	}
	if req == nil || req.GetRecordId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "record_id required")
	}

	rec, err := s.store.GetRecord(req.GetRecordId())
	if err != nil {
		return nil, status.Error(codes.Internal, "get record failed: "+err.Error())
	}
	if rec == nil {
		return &pb_battle_record.GetBattleRecordRsp{}, nil
	}

	info, err := battle_records.RecordToInfo(rec)
	if err != nil {
		return nil, status.Error(codes.Internal, "decode record failed: "+err.Error())
	}
	return &pb_battle_record.GetBattleRecordRsp{Record: info}, nil
}

// ListBattleRecords 按 tag（角色/联盟/地块）分页查询
func (s *BattleRecordServer) ListBattleRecords(ctx context.Context, req *pb_battle_record.ListBattleRecordsReq) (*pb_battle_record.ListBattleRecordsRsp, error) {
	if s.store == nil {
		return nil, status.Error(codes.Internal, "store not initialized")
	}
	if req == nil || req.GetTagType() == pb_battle_record.TagType_TAG_TYPE_UNKNOWN || req.GetTagId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "valid tag_type and tag_id required")
	}

	recs, total, err := s.store.ListRecords(int32(req.GetTagType()), req.GetTagId(), req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, "list records failed: "+err.Error())
	}

	resp := &pb_battle_record.ListBattleRecordsRsp{Total: int32(total)}
	for _, rec := range recs {
		info, err := battle_records.RecordToInfo(rec)
		if err != nil {
			return nil, status.Error(codes.Internal, "decode record failed: "+err.Error())
		}
		resp.Records = append(resp.Records, info)
	}
	return resp, nil
}
