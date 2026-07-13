package crontabs

import (
	"sync/atomic"

	"github.com/robfig/cron/v3"
)

func AddNotRaceFunc(spec string, cmd func()) (cron.EntryID, error) {
	d := struct {
		f     func()
		valid atomic.Bool
	}{
		f: cmd,
	}
	return Get().AddFunc(spec, func() {
		if !d.valid.CompareAndSwap(false, true) {
			return
		}
		defer d.valid.Store(false)
		d.f()
	})
}

// AddFunc 添加任务
func AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	return Get().AddFunc(spec, cmd)
}
