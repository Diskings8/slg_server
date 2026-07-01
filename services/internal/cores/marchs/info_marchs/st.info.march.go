package info_marchs

import "sync"

type MarchInfo struct {
	rwLock sync.RWMutex
}

func (mi *MarchInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MarchInfo) Unlock() {
	mi.rwLock.Unlock()
}
