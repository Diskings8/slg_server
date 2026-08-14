package map_events

import "time"

// EventType 地图地块事件类型
type EventType int32

const (
	// EventTypeUnknown 未知/未定义
	EventTypeUnknown EventType = iota
	// 后续按需扩展：资源采集 / 怪物营 / 宝箱奇遇 / 陷阱 ...
)

// OverlayEvent 地图事件覆盖层数据，表示地图格子上触发的事件信息。
//
// 事件由业务方（如"审查"玩法）刷到主城周围的地块上，扫荡行军到达时
// 校验 MarchInfo.TargetEventID 与 EventID 一致后按 EventType 分派处理。
// 事件为内存态（不随 map_data 持久化，重启后需重新刷出）。
type OverlayEvent struct {
	EventID    uint64    // 事件唯一ID（扫荡创建行军时记录进 MarchInfo.TargetEventID，到达时校对）
	EventType  EventType // 事件类型（处理器注册表按此分派）
	ExpireTime int64     // 过期时间（unix 秒），到达前已过期视为无事件（回退守军）
	Data       any       // 事件附加数据（按类型自定义）
}

func (oe *OverlayEvent) AfterFree(freeTime time.Time) {
	if oe == nil {
		return
	}
	// TODO: 事件过期/清理逻辑（如从地块移除、通知）
}
