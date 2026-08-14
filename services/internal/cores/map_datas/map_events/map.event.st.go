package map_events

import "time"

// EventType 地图地块事件类型
type EventType int32

const (
	// EventTypeUnknown 未知/未定义
	EventTypeUnknown EventType = iota
	// EventTypeResource 采集（资源地，气泡点击）
	EventTypeResource
	// EventTypeMonster 打怪（怪物营/精英怪，扫荡行军）
	EventTypeMonster
	// EventTypeTreasure 寻宝（宝箱奇遇，气泡点击）
	EventTypeTreasure
)

// EventInteraction 事件交互方式
type EventInteraction int32

const (
	// EventInteractionMarch 行军类（打怪，需扫荡行军处理）
	EventInteractionMarch EventInteraction = 0
	// EventInteractionClick 气泡点击类（采集/寻宝，点击 +进度）
	EventInteractionClick EventInteraction = 1
)

// EventClickProgressStep 每次气泡点击增加的进度（固定 26%，超 100% 完成）
const EventClickProgressStep = 26

// Interaction 事件类型对应的交互方式（寻宝/采集=点击；打怪=行军）
func (t EventType) Interaction() EventInteraction {
	switch t {
	case EventTypeMonster:
		return EventInteractionMarch
	default: // Resource / Treasure
		return EventInteractionClick
	}
}

// OverlayEvent 地图事件覆盖层数据，表示地图格子上触发的事件信息。
//
// 事件由业务方（如"审查"玩法）刷到主城周围的地块上，扫荡行军到达时
// 校验 MarchInfo.TargetEventID 与 EventID 一致后按 EventType 分派处理。
// 事件为内存态（不随 map_data 持久化，重启后需重新刷出）。
type OverlayEvent struct {
	EventID      uint64           // 事件唯一ID（扫荡创建行军时记录进 MarchInfo.TargetEventID，到达时校对）
	EventType    EventType        // 事件类型（处理器注册表按此分派）
	Interaction  EventInteraction // 交互方式（行军/气泡点击）
	Progress     int32            // 气泡点击进度 0~100+（CLICK 用；超 100% 完成）
	ExpireTime   int64            // 过期时间（unix 秒），到达前已过期视为无事件（回退守军）
	Data         any              // 事件附加数据（按类型自定义）
}

func (oe *OverlayEvent) AfterFree(freeTime time.Time) {
	if oe == nil {
		return
	}
	// TODO: 事件过期/清理逻辑（如从地块移除、通知）
}
