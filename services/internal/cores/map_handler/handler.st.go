package map_handler

import (
	"server.slg.com/services/internal/cores/map_datas"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// MarchHandler 行军操作 handler
//
// 职责：请求校验 + 编排，不直接操作持久层。
// 所有数据操作委托给 MapManager 及其子管理器。
//
// 函数字段注入模式：
//
//	handler := &MarchHandler{
//	    Manage: func() *map_managers.MapManager { return mm },
//	}
type MarchHandler struct {
	Manage func() *map_managers.MapManager
	Map    func() *map_datas.MapDataManager
	March  func() *marchs.MarchInfoManager
}

// MapHandler 地图操作 handler
//
// 职责：地图相关操作（主城迁移、地块查询等）的校验与编排。
type MapHandler struct {
	Manage func() *map_managers.MapManager
	Map    func() *map_datas.MapDataManager
	March  func() *marchs.MarchInfoManager
}
