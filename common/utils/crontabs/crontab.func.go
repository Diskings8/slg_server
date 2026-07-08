package crontabs

import (
	"github.com/robfig/cron/v3"
	"sync/atomic"
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
