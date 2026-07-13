package crontabs

import (
	"time"

	"github.com/robfig/cron/v3"
)

var crontab = cron.New(cron.WithSeconds(), cron.WithLocation(time.Local))

// Get 获取全局实例
func Get() *cron.Cron {
	return crontab
}

// Start 执行全局实例，非阻塞
func Start() {
	Get().Start()
}

// ShutDown 停止全局实例，非阻塞
func ShutDown() {
	<-Get().Stop().Done()
}
