package common_declarations

import "time"

const (
	DbTypeMysql = "mysql"
	// CaCheKEY 轮询器缓存键
	CaCheKEY = "pollers"
	// CaCheQueueKEY 轮询器保存队列键
	CaCheQueueKEY = "pollers_queue"
)

const (
	// PollerLockTimeout 获取数据等待时间，超时报错
	PollerLockTimeout = 1 * time.Second
	// PollerLongLockTimeout 角色锁定时间过长，强行解锁
	PollerLongLockTimeout = 3 * time.Second

	// PollerCleanupInterval 轮询器清理间隔
	PollerCleanupInterval = time.Minute

	// PollerInactiveTimeout 轮询器不活跃超时时间
	PollerInactiveTimeout = time.Hour
)
