// Package troop 兵种配置表（Go 内嵌占位数据，后续可迁 JSON）
package troop

import (
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/common_declarations"
)

// Conf 兵种配置聚合
type Conf struct {
	TransformLevel uint32   // 转化等级门槛（x 级后可转化派生类型）
	DefaultTroopID int32    // 英雄默认基础兵种
	UnlockItemConf int32    // 扩展兵种所需道具配置ID
	TransformCost  []common_declarations.ItemUse // 兵种转化消耗（资源混搭，走 ItemChange）

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造兵种配置（内置占位数据）
func New() *Conf {
	return &Conf{
		TransformLevel: 10,
		DefaultTroopID: 100,
		UnlockItemConf: 1001,
		TransformCost: []common_declarations.ItemUse{
			{ItemID: pb_confs.ResourceWoodConfID, ItemType: pb_confs.ItemTypeResource, Count: 300},
			{ItemID: pb_confs.ResourceStoneConfID, ItemType: pb_confs.ItemTypeResource, Count: 300},
			{ItemID: pb_confs.ResourceFoodConfID, ItemType: pb_confs.ItemTypeResource, Count: 200},
			{ItemID: pb_confs.ResourceIronConfID, ItemType: pb_confs.ItemTypeResource, Count: 150},
		},
	}
}
