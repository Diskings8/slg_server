package game_logics

import (
	"time"

	"server.slg.com/api/game_conf"
	"server.slg.com/api/game_conf/resource"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
	"server.slg.com/services/game/game_entitys/game_roles"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// resourceBaseCap 资源基础上限：未建仓库时也保底可存（后续可迁 building 配置）
const resourceBaseCap int64 = 100000

// init 注入资源上限计算：实体层 AddItem 对 ItemTypeResource 钳制时调用
func init() {
	game_roles.SetResourceCapFunc(ResourceCap)
}

// ResourceCap 资源存量上限 = 基础上限 + 仓库建筑等级 × CapPerLevel
//
// 仓库升级中（Constructing）沿用旧等级加成（与兵营一致：升级中旧等级继续生效）。
func ResourceCap(r *game_roles.Role, configID pb_confs.ItemID) int64 {
	conf, ok := game_conf.Load().Building.GetBuilding(pb_city.BuildingType_RoleWarehouse)
	if !ok {
		return resourceBaseCap
	}
	capPerLevel := int64(conf.CapPerLevel)
	if capPerLevel <= 0 {
		return resourceBaseCap
	}
	return resourceBaseCap + warehouseLevel(r)*capPerLevel
}

// warehouseLevel 仓库建筑等级（多城取最高等级）
func warehouseLevel(r *game_roles.Role) int64 {
	var lvl int64
	for _, b := range r.GetBuildings().List {
		if b.Type != pb_city.BuildingType_RoleWarehouse {
			continue
		}
		if int64(b.Level) > lvl {
			lvl = int64(b.Level)
		}
	}
	return lvl
}

// ------------------------------- 资源地产出结算 -------------------------------

// elementToResourceID 地块元素类型(Resources_1~4) → 资源道具ConfigID
func elementToResourceID(et cores_declarations.ElementType) pb_confs.ItemID {
	switch et {
	case cores_declarations.ElementType_Resources_1:
		return pb_confs.ResourceFoodConfID
	case cores_declarations.ElementType_Resources_2:
		return pb_confs.ResourceWoodConfID
	case cores_declarations.ElementType_Resources_3:
		return pb_confs.ResourceStoneConfID
	case cores_declarations.ElementType_Resources_4:
		return pb_confs.ResourceIronConfID
	}
	return 0
}

// isResourceElement 是否资源元素（Resources_1~4）
func isResourceElement(et cores_declarations.ElementType) bool {
	return et >= cores_declarations.ElementType_Resources_1 &&
		et <= cores_declarations.ElementType_Resources_4
}

// tileResourceOutputs 地块每小时产出的资源列表（按 resource.conf 等级类型）：
//
//   - lv1 Mixed：全 4 项各产 Amount
//   - lv2 Dual：主资源（元素类型）PrimaryAmount + 次级资源 SecondaryAmount（次级按 mapID 稳定派生）
//   - lv3~9 Single：只产主资源（元素类型）Amount
func tileResourceOutputs(mapID int32, level int32, et cores_declarations.ElementType) []common_declarations.ItemUse {
	cfg := game_conf.Load().Resource.GetConfig(level)
	if cfg == nil {
		return nil
	}
	mainID := elementToResourceID(et)
	if mainID == 0 {
		return nil
	}
	switch resource.ResourceType(cfg.Type) {
	case resource.ResourceTypeMixed:
		// lv1 混合型：全 4 项各产 Amount
		amt := int64(cfg.Amount)
		return []common_declarations.ItemUse{
			{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceFoodConfID, Count: amt},
			{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceWoodConfID, Count: amt},
			{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceStoneConfID, Count: amt},
			{ItemType: pb_confs.ItemTypeResource, ItemID: pb_confs.ResourceIronConfID, Count: amt},
		}
	case resource.ResourceTypeDual:
		// lv2 双资源：主资源 PrimaryAmount + 次级 SecondaryAmount
		sec := dualSecondaryResourceID(mainID, mapID)
		return []common_declarations.ItemUse{
			{ItemType: pb_confs.ItemTypeResource, ItemID: mainID, Count: int64(cfg.PrimaryAmount)},
			{ItemType: pb_confs.ItemTypeResource, ItemID: sec, Count: int64(cfg.SecondaryAmount)},
		}
	default: // Single
		return []common_declarations.ItemUse{
			{ItemType: pb_confs.ItemTypeResource, ItemID: mainID, Count: int64(cfg.Amount)},
		}
	}
}

// dualSecondaryResourceID lv2 双资源的次级资源：按 mapID 稳定派生，保证结算确定性。
// 结果恒 ≠ 主资源（偏移 1~3）。
func dualSecondaryResourceID(mainID pb_confs.ItemID, mapID int32) pb_confs.ItemID {
	mainIdx := int32(mainID) - int32(pb_confs.ResourceFoodConfID) // 0~3
	secIdx := (mainIdx + int32(mapID)%3 + 1) % 4
	return pb_confs.ItemID(int32(pb_confs.ResourceFoodConfID) + secIdx)
}

// SettleRoleResources 惰性结算资源地产出：elapsed(秒) × 每小时产量 → AddResource（cap 钳制）。
//
// 与建筑惰性结算同模式：登录/查询/变更时调用，把上次结算到现在的产出入账，
// 每块地结算后重置 LastSettleUx=now。返回是否有产出入账（调用方决定 poller.Save）。
func SettleRoleResources(role *game_roles.Role, roleID uint64) bool {
	now := time.Now().Unix()
	changed := false
	for _, tile := range role.GetResourceTiles().List {
		if !isResourceElement(cores_declarations.ElementType(tile.ElementType)) {
			continue // 非资源地不产出
		}
		if tile.LastSettleUx <= 0 {
			tile.LastSettleUx = now // 首次同步：只记起点，不补产
			changed = true
			continue
		}
		elapsed := now - tile.LastSettleUx
		if elapsed <= 0 {
			continue
		}
		outputs := tileResourceOutputs(tile.MapID, tile.Level, cores_declarations.ElementType(tile.ElementType))
		if len(outputs) == 0 {
			tile.LastSettleUx = now
			continue
		}
		produced := false
		for _, o := range outputs {
			amount := o.Count * elapsed / 3600
			if amount <= 0 {
				continue
			}
			role.AddItem([]common_declarations.ItemUse{
				{ItemType: pb_confs.ItemTypeResource, ItemID: o.ItemID, Count: amount},
			}, "settle", string(common_declarations.ReasonReward), now)
			produced = true
		}
		// 无论是否有产出都推进计时点（避免亚小时残值反复累积）
		tile.LastSettleUx = now
		if produced {
			changed = true
		}
	}
	return changed
}

// SyncResourceTile 地块产出快照同步（worldmap 事件驱动：开发升级/攻占/放弃）。
//
// 先按旧状态惰性结算到当前时刻，再更新地块状态（等级变更不误算整段旧速率）；
// 目标非资源元素 → 移除快照（不再产出）。
func SyncResourceTile(role *game_roles.Role, roleID uint64, mapID int32, level int32, elementType int32) {
	now := time.Now().Unix()
	if isResourceElement(cores_declarations.ElementType(elementType)) {
		SettleRoleResources(role, roleID) // 旧状态结算到 now
		role.GetResourceTiles().Upsert(roleID, mapID, level, elementType, now)
	} else {
		role.GetResourceTiles().Remove(mapID)
	}
}
