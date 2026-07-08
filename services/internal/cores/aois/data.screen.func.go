package aois

import (
	"sync/atomic"

	"server.slg.com/common/utils/maths"
	"server.slg.com/services/internal/cores/cores_declarations"
)

func NewAoi(mapConfig cores_declarations.MapConfigI) *ScreenData {
	tmp := ScreenData{
		mapConf:         mapConfig,
		mapScope:        mapConfig.MapScope(),
		screenScopeHalf: cores_declarations.ScreenWeight,
		scopeCount:      mapConfig.MapScope() / cores_declarations.ScreenWeight,
	}
	tmp.count = tmp.scopeCount * tmp.scopeCount
	tmp.data = make([]*Screen[cores_declarations.ScreenID], tmp.count+1)
	for screenID := range tmp.data {
		tmp.data[screenID] = &Screen[cores_declarations.ScreenID]{
			ID:     cores_declarations.ScreenID(screenID),
			around: &atomic.Pointer[[]*Screen[cores_declarations.ScreenID]]{},
		}
	}
	return &tmp
}

// AroundByScreen 返回9宫格视野块
func (sd *ScreenData) AroundByScreen(screen *Screen[cores_declarations.ScreenID]) []*Screen[cores_declarations.ScreenID] {
	if screen.around.Load() != nil {
		// 使用已缓存的数据
		return *screen.around.Load()
	}
	// 务必返回9个数据
	tmp := make([]*Screen[cores_declarations.ScreenID], 0, 9)
	tmp = append(tmp, screen) // 中心数据

	var screenIDs []int32
	var useScreenID = int32(screen.ID)

	switch useScreenID % sd.scopeCount {
	case 1: // 最左边
		// 要屏蔽左边
		screenIDs = []int32{
			useScreenID - sd.scopeCount,     // 上
			useScreenID + sd.scopeCount,     // 下
			0,                               // 左
			useScreenID + 1,                 // 右
			0,                               // 上左
			0,                               // 下左
			useScreenID - sd.scopeCount + 1, // 上右
			useScreenID + sd.scopeCount + 1, // 下右
		}
	case 0: // 最右边
		// 要屏蔽右边
		screenIDs = []int32{
			useScreenID - sd.scopeCount,     // 上
			useScreenID + sd.scopeCount,     // 下
			useScreenID - 1,                 // 左
			0,                               // 右
			useScreenID - sd.scopeCount - 1, // 上左
			useScreenID + sd.scopeCount - 1, // 下左
			0,                               // 上右
			0,                               // 下右
		}
	default:
		screenIDs = []int32{
			useScreenID - sd.scopeCount,     // 上
			useScreenID + sd.scopeCount,     // 下
			useScreenID - 1,                 // 左
			useScreenID + 1,                 // 右
			useScreenID - sd.scopeCount - 1, // 上左
			useScreenID + sd.scopeCount - 1, // 下左
			useScreenID - sd.scopeCount + 1, // 上右
			useScreenID + sd.scopeCount + 1, // 下右
		}
	}

	// 填入nil保证返回数量9
	for _, screenID := range screenIDs {
		if screenID > 0 && screenID < sd.count {
			tmp = append(tmp, screen)
		} else {
			tmp = append(tmp, nil)
		}
	}
	return tmp
}

func (sd *ScreenData) Cover(mapID cores_declarations.MapID, cover int32) []*Screen[cores_declarations.ScreenID] {
	var out []*Screen[cores_declarations.ScreenID]
	screenData := sd.GetScreenByMapID(mapID)
	if cover == 0 {
		return []*Screen[cores_declarations.ScreenID]{screenData}
	}

	var useMapID = int32(mapID)

	baseX := useMapID % sd.mapScope / sd.screenScopeHalf
	baseY := useMapID / sd.mapScope / sd.screenScopeHalf

	startX := max(maths.Int32(baseX, -cover), 0)
	endX := min(maths.Int32(baseX, cover), sd.scopeCount)
	startY := max(maths.Int32(baseY, -cover), 0)
	endY := min(maths.Int32(baseY, cover), sd.scopeCount)
	for x := startX; x <= endX; x++ {
		for y := startY; y <= endY; y++ {
			screenID := x + y*sd.scopeCount + 1
			if screenID > sd.count {
				continue
			}
			out = append(out, sd.GetScreenByScreenID(cores_declarations.ScreenID(screenID)))
		}
	}
	return out
}

func (sd *ScreenData) Around(mapID cores_declarations.MapID) []*Screen[cores_declarations.ScreenID] {
	return sd.AroundByScreen(sd.GetScreenByMapID(mapID))
}

func (sd *ScreenData) AroundConnects(mapID cores_declarations.MapID, connects map[uint64]cores_declarations.MapRoleConnectI) map[uint64]cores_declarations.MapRoleConnectI {
	for _, v := range sd.Around(mapID) {
		v.Connects(connects)
	}
	return connects
}

//------------------------------Connect---------------------------//

func (sd *ScreenData) Exit(roleConn cores_declarations.MapRoleConnectI) {
	screen := sd.GetScreenByMapID(roleConn.GetScreenMapID())
	screen.connect.Delete(roleConn.GetRoleID())
}

//--------------------------------

// Move 视野移动
func (sd *ScreenData) Move(conn cores_declarations.MapRoleConnectI, newMapID cores_declarations.MapID) {
	oldScreenID := sd.GetScreenIDByMapID(conn.GetScreenMapID())
	newScreenID := sd.GetScreenIDByMapID(newMapID)
	if oldScreenID == newScreenID {
		// 在同一个级别情况下移动不进行处理
		return
	}
	defer conn.SetScreenMapID(newMapID)

	sd.GetScreenByScreenID(oldScreenID).connect.Delete(conn.GetRoleID())
	if newMapID >= 0 {
		sd.GetScreenByScreenID(newScreenID).connect.Store(conn.GetRoleID(), conn)
	}
}

func (sd *ScreenData) MovePath(x int32, y int32, x2 int32, y2 int32, i *[]*Screen[cores_declarations.ScreenID]) []*Screen[cores_declarations.ScreenID] {
	return nil
}
