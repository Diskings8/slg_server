// Package troop 兵种配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
package troop

import "server.slg.com/common/common_declarations"

// Conf 兵种配置聚合
type Conf struct {
	TransformLevel uint32   // 转化等级门槛（x 级后可转化派生类型）
	DefaultTroopID int32    // 英雄默认基础兵种
	UnlockItemConf int32    // 扩展兵种所需道具配置ID
	TransformCost  []common_declarations.ItemUse // 兵种转化消耗（资源混搭，走 ItemChange）
}
