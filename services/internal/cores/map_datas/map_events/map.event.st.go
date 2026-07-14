package map_events

import "time"

// OverlayEvent 地图事件覆盖层数据，表示地图格子上触发的事件信息
type OverlayEvent struct {
}

func (oe *OverlayEvent) AfterFree(freeTime time.Time) {
	if oe == nil {
		return
	}
}
