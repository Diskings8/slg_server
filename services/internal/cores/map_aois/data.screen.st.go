package map_aois

import (
	"go.uber.org/zap"
	"server.slg.com/common/loggers"
	"server.slg.com/services/internal/cores/cores_declarations"
)

// ScreenData AOI（Area of Interest）屏幕格子数据，用于管理场景中的视野区域
type ScreenData struct {
	data            []*Screen[cores_declarations.ScreenID]
	mapConf         cores_declarations.MapConfigI
	mapScope        int32
	screenScopeHalf int32 // 屏幕边长/2
	scopeCount      int32 // 一行数量
	count           int32 // 总数
}

//---------------------------Get-------------------------------------//

// GetScreenIDByMapID 用MapId获取视野ID
func (sd *ScreenData) GetScreenIDByMapID(mapID cores_declarations.MapID) cores_declarations.ScreenID {
	var screenID int32
	var useMapID = int32(mapID)
	if mapID >= 0 && useMapID < sd.mapConf.MapCount() {
		screenID = sd.XY2ScreenID(useMapID%sd.mapScope, useMapID/sd.mapScope)
	}
	return cores_declarations.ScreenID(screenID)
}

// GetScreenByMapID 用MapId获取视野信息
func (sd *ScreenData) GetScreenByMapID(mapID cores_declarations.MapID) *Screen[cores_declarations.ScreenID] {
	return sd.GetScreenByScreenID(sd.GetScreenIDByMapID(mapID))
}

// GetScreenByScreenID 用视野id获取视野信息
func (sd *ScreenData) GetScreenByScreenID(screenID cores_declarations.ScreenID) *Screen[cores_declarations.ScreenID] {
	if int32(screenID) > sd.count {
		loggers.Logger.Error("screenID error", zap.Int32("screenID", int32(screenID)))
		screenID = 0
	}
	return sd.data[screenID]
}

// GetConnects 获取MapId的链接
func (sd *ScreenData) GetConnects(mapID cores_declarations.MapID, connects map[uint64]cores_declarations.MapRoleConnectI) map[uint64]cores_declarations.MapRoleConnectI {
	return sd.GetScreenByMapID(mapID).Connects(connects)
}

// GetMapIDFirstByScreenID 取得视野切块的第一个坐标
func (sd *ScreenData) GetMapIDFirstByScreenID(screenID cores_declarations.ScreenID) cores_declarations.MapID {
	useScreenID := int32(screenID)
	x := ((useScreenID - 1) % sd.scopeCount) * sd.screenScopeHalf
	y := ((useScreenID - 1) / sd.scopeCount) * sd.screenScopeHalf
	return sd.mapConf.XY2MapID(x, y)
}

// ScreenIDs2MapIDs 取得视野切块的所有MapID
func (sd *ScreenData) ScreenIDs2MapIDs(screenIDs []cores_declarations.ScreenID, mapIDs *[]cores_declarations.MapID) {
	var x, y int32
	for _, screenID := range screenIDs {
		useScreenID := int32(screenID)
		x = ((useScreenID - 1) % sd.scopeCount) * sd.screenScopeHalf
		y = ((useScreenID - 1) / sd.scopeCount) * sd.screenScopeHalf
		for yAdd := int32(0); yAdd < sd.screenScopeHalf; yAdd++ {
			for xAdd := int32(0); xAdd < sd.screenScopeHalf; xAdd++ {
				*mapIDs = append(*mapIDs, sd.mapConf.XY2MapID(x+xAdd, y+yAdd))
			}
		}
	}
}

func (sd *ScreenData) MapID2ScreenID(mapID cores_declarations.MapID) cores_declarations.ScreenID {
	var useMapID = int32(mapID)
	var screenID int32
	if mapID >= 0 && int32(mapID) < sd.mapConf.MapCount() {
		screenID = sd.XY2ScreenID(useMapID%sd.mapScope, useMapID/sd.mapScope)
	}
	return cores_declarations.ScreenID(screenID)
}

// ScreenMapLen 取得视野切块数量
func (sd *ScreenData) ScreenMapLen() int {
	return int(sd.screenScopeHalf * sd.screenScopeHalf)
}

// XY2ScreenID 取得视野切块ID
func (sd *ScreenData) XY2ScreenID(x, y int32) int32 {
	if (x < 0 || x >= sd.mapConf.MapScope()) || (y < 0 || y >= sd.mapConf.MapScope()) {
		return 0
	}
	return x/sd.screenScopeHalf + y/sd.screenScopeHalf*sd.scopeCount + 1
}
