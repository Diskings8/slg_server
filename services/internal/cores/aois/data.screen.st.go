package aois

import "server.slg.com/services/internal/cores/cores_declarations"

// ScreenData AOI（Area of Interest）屏幕格子数据，用于管理场景中的视野区域
type ScreenData struct {
}

func (sd *ScreenData) MapDataAdd(mapID cores_declarations.MapID) {

}

func (sd *ScreenData) GetScreen(id cores_declarations.MarchID) *Screen[int32] {
	return nil
}

func (sd *ScreenData) MovePath(x int32, y int32, x2 int32, y2 int32, i *[]*Screen[int32]) []*Screen[int32] {
	return nil
}
