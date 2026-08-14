package battle_records

// 战报存储 — 纯 MySQL。
// 主表只存记录，查询走 battle_record_tag 索引表（单复合索引覆盖角色/联盟/地块三维）。
// 14 天保留，节点内定时清理。

import (
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"
	"server.slg.com/api/protocol/pb/pb_battle"
	"server.slg.com/api/protocol/pb/pb_battle_record"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/services/battle_record/battle_record_models"

	"google.golang.org/protobuf/proto"
)

// 战报保留时长：14 天
const RetentionDays = 14

// 查询维度 TagType（对齐 proto battle_record.TagType）
const (
	TagTypeRole  = int32(pb_battle_record.TagType_TAG_TYPE_ROLE)
	TagTypeUnion = int32(pb_battle_record.TagType_TAG_TYPE_UNION)
	TagTypeTile  = int32(pb_battle_record.TagType_TAG_TYPE_TILE)
)

// Store 战报存储
type Store struct {
	db *gorm.DB
}

// New 创建战报存储
func New(db *gorm.DB) *Store {
	return &Store{db: db}
}

// Migrate 建表 + 索引（AutoMigrate 幂等，索引由 model tag 声明）
func (s *Store) Migrate() error {
	return s.db.AutoMigrate(&battle_record_models.BattleRecord{}, &battle_record_models.BattleRecordTag{})
}

// SaveRecord 保存战报：事务写入主表 + 索引表
//
// ModelBase.ID 非自增，由应用层雪花 ID 生成（对齐 game 建筑等实体）。
func (s *Store) SaveRecord(rec *battle_record_models.BattleRecord) error {
	if rec.ID == 0 {
		rec.ID = snowflakes.GenUUID()
	}

	tags := buildTags(rec)
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(rec).Error; err != nil {
			return err
		}
		if len(tags) > 0 {
			if err := tx.CreateInBatches(tags, 100).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetRecord 按战报 ID 查询，未找到返回 nil
func (s *Store) GetRecord(id uint64) (*battle_record_models.BattleRecord, error) {
	var rec battle_record_models.BattleRecord
	if err := s.db.Where("id = ?", id).First(&rec).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

// ListRecords 按 tag 分页查询（battle_time 倒序）
func (s *Store) ListRecords(tagType int32, tagID uint64, page, pageSize int32) ([]*battle_record_models.BattleRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := s.db.Model(&battle_record_models.BattleRecordTag{}).
		Where("tag_type = ? AND tag_id = ?", tagType, tagID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var tagRows []*battle_record_models.BattleRecordTag
	offset := (page - 1) * pageSize
	if err := s.db.Model(&battle_record_models.BattleRecordTag{}).
		Where("tag_type = ? AND tag_id = ?", tagType, tagID).
		Order("battle_time DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&tagRows).Error; err != nil {
		return nil, 0, err
	}

	if len(tagRows) == 0 {
		return nil, total, nil
	}

	ids := make([]uint64, 0, len(tagRows))
	for _, t := range tagRows {
		ids = append(ids, t.BattleRecordID)
	}
	var recs []*battle_record_models.BattleRecord
	if err := s.db.Where("id IN ?", ids).Find(&recs).Error; err != nil {
		return nil, 0, err
	}

	// 按 tag 行顺序（battle_time DESC）重排
	order := make(map[uint64]int, len(tagRows))
	for i, t := range tagRows {
		order[t.BattleRecordID] = i
	}
	sort.Slice(recs, func(i, j int) bool {
		return order[recs[i].ID] < order[recs[j].ID]
	})
	return recs, total, nil
}

// ListChildRecords 查询主战报的子战报（车轮战 n 队整合），battle_time 倒序分页
func (s *Store) ListChildRecords(parentID uint64, page, pageSize int32) ([]*battle_record_models.BattleRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int64
	if err := s.db.Model(&battle_record_models.BattleRecord{}).
		Where("parent_id = ?", parentID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var recs []*battle_record_models.BattleRecord
	offset := (page - 1) * pageSize
	if err := s.db.Where("parent_id = ?", parentID).
		Order("battle_time DESC").
		Limit(int(pageSize)).
		Offset(int(offset)).
		Find(&recs).Error; err != nil {
		return nil, 0, err
	}
	return recs, total, nil
}

// CleanupExpired 清理 battle_time 早于 cutoff 的过期战报（两张表）
func (s *Store) CleanupExpired(cutoff int64) error {
	if err := s.db.Where("battle_time < ?", cutoff).Delete(&battle_record_models.BattleRecord{}).Error; err != nil {
		return err
	}
	return s.db.Where("battle_time < ?", cutoff).Delete(&battle_record_models.BattleRecordTag{}).Error
}

// buildTags 由主表记录生成查询索引行（攻守双方 × role/union + tile，去重）。
// 子战报（parent_id != 0）不生成 tag —— 只通过主战报进入，避免玩家列表重复。
func buildTags(rec *battle_record_models.BattleRecord) []*battle_record_models.BattleRecordTag {
	if rec.ParentID != 0 {
		return nil
	}

	type tagKey struct {
		t  int32
		id uint64
	}
	seen := make(map[tagKey]struct{})
	var tags []*battle_record_models.BattleRecordTag

	add := func(t int32, id uint64) {
		if id == 0 {
			return
		}
		k := tagKey{t, id}
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		tags = append(tags, &battle_record_models.BattleRecordTag{
			TagType:        t,
			TagID:          id,
			BattleRecordID: rec.ID,
			BattleTime:     rec.BattleTime,
		})
	}

	add(TagTypeRole, rec.AttackerRoleID)
	add(TagTypeUnion, rec.AttackerUnionID)
	if rec.MapID >= 0 {
		add(TagTypeTile, uint64(rec.MapID))
	}
	for _, id := range rec.DefenderRoleIDs {
		add(TagTypeRole, id)
	}
	for _, id := range rec.DefenderUnionIDs {
		add(TagTypeUnion, id)
	}
	return tags
}

// RecordFromReq SaveBattleRecordReq → 主表 model
// nonNilUint64Slice 空/nil 切片归一为空数组（json.Marshal(nil) → null/空串，MySQL JSON 列会拒收空文档）。
func nonNilUint64Slice(s []uint64) []uint64 {
	if s == nil {
		return []uint64{}
	}
	return s
}

func RecordFromReq(req *pb_battle_record.SaveBattleRecordReq) (*battle_record_models.BattleRecord, error) {
	battleTime := req.GetBattleTime()
	if battleTime == 0 {
		battleTime = time.Now().Unix()
	}

	rec := &battle_record_models.BattleRecord{
		MarchID:          req.GetMarchId(),
		MarchType:        req.GetMarchType(),
		MapID:            req.GetMapId(),
		AttackerRoleID:   req.GetAttackerRoleId(),
		AttackerUnionID:  req.GetAttackerUnionId(),
		DefenderRoleIDs:  nonNilUint64Slice(req.GetDefenderRoleIds()),
		DefenderUnionIDs: nonNilUint64Slice(req.GetDefenderUnionIds()),
		AttackerWin:      req.GetAttackerWin(),
		IsOccupied:       req.GetIsOccupied(),
		BuildingDamage:   req.GetBuildingDamage(),
		BattleTime:       battleTime,
		ParentID:         req.GetParentId(),
	}

	if req.GetResults() != nil {
		data, err := proto.Marshal(req.GetResults())
		if err != nil {
			return nil, err
		}
		rec.Results = data
	}
	return rec, nil
}

// RecordToInfo 主表 model → pb BattleRecordInfo
func RecordToInfo(rec *battle_record_models.BattleRecord) (*pb_battle_record.BattleRecordInfo, error) {
	info := &pb_battle_record.BattleRecordInfo{
		RecordId:         rec.ID,
		MarchId:          rec.MarchID,
		AttackerRoleId:   rec.AttackerRoleID,
		AttackerUnionId:  rec.AttackerUnionID,
		DefenderRoleIds:  rec.DefenderRoleIDs,
		DefenderUnionIds: rec.DefenderUnionIDs,
		MapId:            rec.MapID,
		MarchType:        rec.MarchType,
		AttackerWin:      rec.AttackerWin,
		IsOccupied:       rec.IsOccupied,
		BuildingDamage:   rec.BuildingDamage,
		BattleTime:       rec.BattleTime,
		ParentId:         rec.ParentID,
	}

	if len(rec.Results) > 0 {
		results := &pb_battle.BattleResults{}
		if err := proto.Unmarshal(rec.Results, results); err != nil {
			return nil, err
		}
		info.Results = results
	}
	return info, nil
}
