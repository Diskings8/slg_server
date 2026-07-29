package role_items

import (
	"errors"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_error_code"
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

var (
	// ErrItemNotEnough 道具数量不足
	ErrItemNotEnough = errors.New("item count not enough")
	// ErrItemNotFound 道具不存在
	ErrItemNotFound = errors.New("item not found")
)

// NewRoleItems 创建角色道具集合
func NewRoleItems(roleID uint64) *RoleItems {
	return &RoleItems{
		RoleID: roleID,
		List:   make([]*game_models.RoleItem, 0),
	}
}

// Init 初始化 Mem 索引
func (ris *RoleItems) Init() {
	for _, modelOne := range ris.List {
		roleItem := NewRoleItem(modelOne)
		ris.Mem.Store(pb_confs.ItemID(roleItem.ConfigID), roleItem)
	}
}

// Copy 深拷贝
func (ris *RoleItems) Copy(src *RoleItems) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, ris)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	ris.Init()
}

// Format2Pb 转换为 protobuf 列表
func (ris *RoleItems) Format2Pb() []*pb_item.ItemUse {
	list := make([]*pb_item.ItemUse, 0, len(ris.List))
	for _, v := range ris.List {
		item := NewRoleItem(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

// -------------------------------

// NewRoleItem 包装单个道具
func NewRoleItem(one *game_models.RoleItem) *RoleItem {
	return &RoleItem{
		RoleItem: one,
	}
}

func (ri *RoleItem) Format2Pb() *pb_item.ItemUse {
	return &pb_item.ItemUse{
		ConfId: int32(ri.ConfigID),
		Count:  ri.Count,
	}
}

// ------------------------------- 数据查询 -------------------------------

// GetItemCount 获取指定道具的数量
func (ris *RoleItems) GetItemCount(configID int32) int64 {
	if exist, ok := ris.Mem.Load(pb_confs.ItemID(configID)); ok {
		return exist.Count
	}
	return 0
}

// HasItem 检查是否有足够数量的道具
func (ris *RoleItems) HasItem(configID int32, count int64) bool {
	return ris.GetItemCount(configID) >= count
}

// CheckItemEnough 检查道具是否充足
func (ris *RoleItems) CheckItemEnough(use common_declarations.ItemUse) pb_error_code.ErrorCode {
	if ris.HasItem(int32(use.ItemID), use.Count) {
		return pb_error_code.ErrorCode_NoneErr
	}
	return pb_error_code.ErrorCode_ItemTypeNormalNotEnough
}

// AddItem 添加道具
func (ris *RoleItems) AddItem(use common_declarations.ItemUse, createUx int64) (curCount int64) {
	item, exit := ris.Mem.Load(use.ItemID)
	if exit {
		item.Count += use.Count
		return item.Count
	}
	modelOne := game_models.NewRoleItem(ris.RoleID, use, snowflakes.GenUUID(), createUx)
	newOne := NewRoleItem(modelOne)
	ris.List = append(ris.List, modelOne)
	ris.Mem.Store(use.ItemID, newOne)
	return newOne.Count
}

// DeleteItem 从 List 和 Mem 中删除指定道具 (慎用
func (ris *RoleItems) DeleteItem(configID pb_confs.ItemID) {
	ris.Mem.Delete(configID)

	for i, v := range ris.List {
		if v.ConfigID == configID {
			ris.List = append(ris.List[:i], ris.List[i+1:]...)
			break
		}
	}
}

// ReduceItem 减少道具数量（不做是否足够检测
func (ris *RoleItems) ReduceItem(configID pb_confs.ItemID, count int64) (curCount int64) {
	item, exit := ris.Mem.Load(configID)
	if exit && item.Count > count {
		item.Count -= count
		return item.Count
	} else if exit {
		item.Count = 0
		return item.Count
	}
	return 0
}
