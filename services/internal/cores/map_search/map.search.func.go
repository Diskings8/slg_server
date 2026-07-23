package map_search

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
)

// MapSearch 地图搜索器，基于 AOI 系统实现近邻格子搜索
type MapSearch struct {
	mm *map_managers.MapManager
}

// New 创建地图搜索器
func New(mm *map_managers.MapManager) *MapSearch {
	return &MapSearch{mm: mm}
}

// SearchOption 搜索参数
type SearchOption struct {
	CenterMapID cores_declarations.MapID       // 中心格子 ID
	MaxRange    int32                          // 搜索范围（格子数），0 表示使用 AOI 九宫格
	ElementType cores_declarations.ElementType // 筛选元素类型，ElementType_None 表示不过滤
	Level       cores_declarations.MapLevel    // 筛选等级，0 表示不过滤
	MaxResult   int                            // 最大返回数量，<=0 表示不限制
}

// SearchResult 搜索结果
type SearchResult struct {
	MapID cores_declarations.MapID
	Type  cores_declarations.ElementType
	Level cores_declarations.MapLevel
}

// NearBy 搜索指定格子附近的格子
//
// 搜索策略：
//   - MaxRange > 0：以中心格子为中心的正方形范围搜索
//   - MaxRange = 0：使用 AOI 九宫格（中心格子所在 Screen 及周围 8 个 Screen）
//
// 返回结果按到中心格子的距离排序（由 MapConfigI.SortByDis 实现）。
func (s *MapSearch) NearBy(opt SearchOption) []SearchResult {
	mdm := s.mm.GetMapDataManager()
	aoi := mdm.AOI
	config := mdm.GetConfig()

	// 收集待检查的 mapID
	checkMapIDs := make(map[cores_declarations.MapID]struct{})

	if opt.MaxRange > 0 {
		// 按范围搜索：以 center 为中心的正方形区域
		cx, cy := config.MapID2XY(opt.CenterMapID)
		startX, endX := cx-opt.MaxRange, cx+opt.MaxRange
		startY, endY := cy-opt.MaxRange, cy+opt.MaxRange

		for x := startX; x <= endX; x++ {
			for y := startY; y <= endY; y++ {
				mapID := config.XY2MapID(x, y)
				if mapID >= 0 {
					checkMapIDs[mapID] = struct{}{}
				}
			}
		}
	} else {
		// AOI 九宫格搜索
		screenIDs := make([]cores_declarations.ScreenID, 0, 9)
		for _, s := range aoi.Around(opt.CenterMapID) {
			if s != nil {
				screenIDs = append(screenIDs, s.ID)
			}
		}
		// 将 ScreenID 列表转换为具体 MapID 列表
		var mapIDs []cores_declarations.MapID
		aoi.ScreenIDs2MapIDs(screenIDs, &mapIDs)
		for _, mapID := range mapIDs {
			checkMapIDs[mapID] = struct{}{}
		}
	}

	// 过滤和筛选
	results := make([]SearchResult, 0, len(checkMapIDs))
	for mapID := range checkMapIDs {
		if mapID == opt.CenterMapID {
			continue // 排除中心格子自身
		}
		if opt.MaxResult > 0 && len(results) >= opt.MaxResult {
			break
		}

		info, ok := mdm.GetMapInfo(mapID)
		if !ok {
			continue
		}

		// 类型过滤
		if opt.ElementType != cores_declarations.ElementType_None {
			if info.GetElementType() != opt.ElementType {
				continue
			}
		}

		// 等级过滤
		if opt.Level > 0 {
			if info.GetLevel() != opt.Level {
				continue
			}
		}

		results = append(results, SearchResult{
			MapID: mapID,
			Type:  info.GetElementType(),
			Level: info.GetLevel(),
		})
	}

	// 按距离排序
	if len(results) > 1 {
		mapIDs := make([]cores_declarations.MapID, len(results))
		for i, r := range results {
			mapIDs[i] = r.MapID
		}
		config.SortByDis(opt.CenterMapID, mapIDs)
		sorted := make([]SearchResult, len(results))
		for i, mid := range mapIDs {
			for _, r := range results {
				if r.MapID == mid {
					sorted[i] = r
					break
				}
			}
		}
		results = sorted
	}

	return results
}

// FreeNearBy 搜索指定格子附近的空闲格子
//
// 等价于 NearBy 加上 ownerID==0 过滤，用于寻找可占据的空地。
func (s *MapSearch) FreeNearBy(opt SearchOption) []SearchResult {
	opt.MaxRange = max(opt.MaxRange, 3) // 空闲搜索至少 3 格范围
	allResults := s.NearBy(opt)
	filtered := make([]SearchResult, 0, len(allResults))

	for _, r := range allResults {
		if opt.MaxResult > 0 && len(filtered) >= opt.MaxResult {
			break
		}
		info, ok := s.mm.GetMapDataManager().GetMapInfo(r.MapID)
		if !ok {
			continue
		}
		// 检查地块是否空闲（无归属、无建筑等）
		if info.GetOwnerID() == 0 && info.GetOverlayBuilding() == nil {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
