package redisstream

// Stream key 前缀
const keyPrefix = "slg:"

// 预定义的 Redis Stream key
const (
	// StreamKeyMarchEvents 行军事件 — worldmap 发布 → game 消费
	StreamKeyMarchEvents = keyPrefix + "march:events"
)
