package game_roles

import (
	"math"

	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// ResourceCapFunc 计算角色某资源类型的存量上限（由 game_logics 注入，避免实体层依赖配置）
type ResourceCapFunc func(r *Role, configID pb_confs.ItemID) int64

var resourceCap ResourceCapFunc

// SetResourceCapFunc 注入资源上限计算函数（game_logics init 时调用）
func SetResourceCapFunc(f ResourceCapFunc) {
	resourceCap = f
}

// resourceCap 查询资源存量上限；未注入计算函数时不过限（实体层测试/独立使用场景）
func (r *Role) resourceCap(configID pb_confs.ItemID) int64 {
	if resourceCap == nil {
		return math.MaxInt64
	}
	return resourceCap(r, configID)
}

// addResource 添加资源并钳制到仓库上限，返回（结算后数量, 实际增加量）。
// 到上限后多余部分丢弃（对应"产出溢出"的 SLG 常规设定）。
func (r *Role) addResource(use common_declarations.ItemUse, optTimeUx int64) (curCount, delta int64) {
	capVal := r.resourceCap(use.ItemID)
	before := r.GetItems().GetItemCount(int32(use.ItemID))
	if before >= capVal {
		return before, 0
	}
	delta = use.Count
	if before+delta > capVal {
		delta = capVal - before
	}
	if delta <= 0 {
		return before, 0
	}
	curCount = r.GetItems().AddItem(common_declarations.ItemUse{
		ItemID: use.ItemID, ItemType: use.ItemType, ItemSubType: use.ItemSubType, Count: delta,
	}, optTimeUx)
	return curCount, delta
}
