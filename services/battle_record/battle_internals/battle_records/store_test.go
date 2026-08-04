package battle_records

// 战报存储单测 — 使用 sqlite 内存库，覆盖：保存 + tag 三维索引 / 分页 / 清理 / 编解码。

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/services/battle_record/battle_record_models"

	"google.golang.org/protobuf/proto"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	// 每个测试用独立内存库，避免数据串扰
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	s := New(db)
	if err := s.Migrate(); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return s
}

func mustMarshalResults(r *pb_battle.BattleResults) []byte {
	data, err := proto.Marshal(r)
	if err != nil {
		panic(err)
	}
	return data
}

// TestSaveAndTagIndexes 保存后 tag 索引覆盖攻守双方 × 三维（去重）
func TestSaveAndTagIndexes(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().Unix()

	rec := &battle_record_models.BattleRecord{
		MarchID:          100,
		MarchType:        10001,
		MapID:            55,
		AttackerRoleID:   1,
		AttackerUnionID:  10,
		DefenderRoleIDs:  []uint64{2, 3},
		DefenderUnionIDs: []uint64{20},
		AttackerWin:      true,
		IsOccupied:       true,
		BuildingDamage:   100,
		Results:          mustMarshalResults(&pb_battle.BattleResults{ResultCount: 1}),
		BattleTime:       now,
	}
	rec.ID = 1000 // 显式指定（单测不初始化 snowflake）
	if err := s.SaveRecord(rec); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// tag 行：role(1,2,3) + union(10,20) + tile(55) = 6 行
	var tags []*battle_record_models.BattleRecordTag
	if err := s.db.Where("battle_record_id = ?", rec.ID).Find(&tags).Error; err != nil {
		t.Fatalf("query tags failed: %v", err)
	}
	if len(tags) != 6 {
		t.Fatalf("期望 6 个 tag，实际 %d", len(tags))
	}

	// 按防守方角色查
	recs, total, err := s.ListRecords(TagTypeRole, 2, 1, 20)
	if err != nil || total != 1 || len(recs) != 1 {
		t.Fatalf("role 查询失败: err=%v total=%d len=%d", err, total, len(recs))
	}
	// 按联盟查
	if _, total, err := s.ListRecords(TagTypeUnion, 20, 1, 20); err != nil || total != 1 {
		t.Fatalf("union 查询失败: err=%v total=%d", err, total)
	}
	// 按地块查
	if _, total, err := s.ListRecords(TagTypeTile, 55, 1, 20); err != nil || total != 1 {
		t.Fatalf("tile 查询失败: err=%v total=%d", err, total)
	}
}

// TestListPagination 分页 + battle_time 倒序
func TestListPagination(t *testing.T) {
	s := newTestStore(t)
	base := time.Now().Unix()
	for i := int64(0); i < 5; i++ {
		rec := &battle_record_models.BattleRecord{
			MarchID: uint64(i + 1), MarchType: 10001, MapID: 60,
			AttackerRoleID: 1, AttackerUnionID: 10,
			AttackerWin: true, Results: []byte{0x0a, 0x00}, // 空 BattleResults
			BattleTime: base + i,
		}
		rec.ID = uint64(i + 1)
		if err := s.SaveRecord(rec); err != nil {
			t.Fatalf("save failed: %v", err)
		}
	}

	// 第 1 页 2 条，倒序 → march_id 5,4
	recs, total, err := s.ListRecords(TagTypeRole, 1, 1, 2)
	if err != nil || total != 5 || len(recs) != 2 {
		t.Fatalf("page1 失败: err=%v total=%d len=%d", err, total, len(recs))
	}
	if recs[0].MarchID != 5 || recs[1].MarchID != 4 {
		t.Fatalf("倒序错误: got %d,%d 期望 5,4", recs[0].MarchID, recs[1].MarchID)
	}

	// 第 3 页 2 条 → march_id 1
	recs, _, err = s.ListRecords(TagTypeRole, 1, 3, 2)
	if err != nil || len(recs) != 1 || recs[0].MarchID != 1 {
		t.Fatalf("page3 失败: err=%v len=%d march=%d", err, len(recs), recs[0].MarchID)
	}
}

// TestCleanupExpired 14 天清理
func TestCleanupExpired(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().Unix()

	old := &battle_record_models.BattleRecord{
		MarchID: 1, MarchType: 10001, MapID: 70,
		AttackerRoleID: 1, AttackerWin: true,
		Results:    []byte{},                        // 非 nil（列 not null）
		BattleTime: now - RetentionDays*24*3600 - 1, // 过期
	}
	old.ID = 1
	newRec := &battle_record_models.BattleRecord{
		MarchID: 2, MarchType: 10001, MapID: 70,
		AttackerRoleID: 1, AttackerWin: true,
		Results:    []byte{}, // 非 nil（列 not null）
		BattleTime: now,      // 未过期
	}
	newRec.ID = 2
	if err := s.SaveRecord(old); err != nil {
		t.Fatalf("save old failed: %v", err)
	}
	if err := s.SaveRecord(newRec); err != nil {
		t.Fatalf("save new failed: %v", err)
	}

	cutoff := now - RetentionDays*24*3600
	if err := s.CleanupExpired(cutoff); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	// 旧记录及其 tag 被删，新记录保留
	if rec, _ := s.GetRecord(old.ID); rec != nil {
		t.Fatalf("过期记录未删除: id=%d", old.ID)
	}
	if rec, _ := s.GetRecord(newRec.ID); rec == nil {
		t.Fatalf("未过期记录被误删")
	}
	var tagCount int64
	s.db.Model(&battle_record_models.BattleRecordTag{}).Where("battle_record_id = ?", old.ID).Count(&tagCount)
	if tagCount != 0 {
		t.Fatalf("过期 tag 未删除: %d", tagCount)
	}
}

// TestRecordFromReqAndToInfo 请求→model→pb 编解码（含 results 二进制）
func TestRecordFromReqAndToInfo(t *testing.T) {
	req := &pb_battle_record.SaveBattleRecordReq{
		MarchId:          100,
		AttackerRoleId:   1,
		AttackerUnionId:  10,
		DefenderRoleIds:  []uint64{2},
		DefenderUnionIds: []uint64{20},
		MapId:            55,
		MarchType:        10001,
		AttackerWin:      true,
		IsOccupied:       true,
		BuildingDamage:   50,
		BattleTime:       123456,
		Results: &pb_battle.BattleResults{
			ResultCount: 2,
			Results:     []*pb_battle.OneBattleResult{{IsOccupied: true}},
		},
	}

	rec, err := RecordFromReq(req)
	if err != nil {
		t.Fatalf("RecordFromReq failed: %v", err)
	}
	if rec.BattleTime != 123456 {
		t.Fatalf("battle_time 期望 123456 实际 %d", rec.BattleTime)
	}
	if len(rec.Results) == 0 {
		t.Fatalf("results 未序列化")
	}

	info, err := RecordToInfo(rec)
	if err != nil {
		t.Fatalf("RecordToInfo failed: %v", err)
	}
	if info.GetResults().GetResultCount() != 2 {
		t.Fatalf("results 回读失败: %d", info.GetResults().GetResultCount())
	}
	if len(info.GetDefenderRoleIds()) != 1 || info.GetDefenderRoleIds()[0] != 2 {
		t.Fatalf("defender_role_ids 回读失败: %v", info.GetDefenderRoleIds())
	}
}
