package map_connects

import (
	"context"
	"sync/atomic"
)

func init() {
	ctx, cancelF := context.WithCancel(context.Background())
	defaultAllConnectManager = &allConnectManager{
		ctx:        ctx,
		cancelFunc: cancelF,
	}
	defaultAllConnectManager.isStop.Store(false)
}

var defaultAllConnectManager *allConnectManager

type allConnectManager struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	isStop     atomic.Bool
}

// ShutDown 全局进程结束
func ShutDown() {
	defaultAllConnectManager.isStop.Store(true)
	defaultAllConnectManager.cancelFunc()
}
