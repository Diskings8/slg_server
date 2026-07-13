package roles

import (
	"slices"
	"sync"

	"server.slg.com/common/common_declarations"
)

func (d *Data) Copy(rw *sync.RWMutex) common_declarations.DataI {
	v := &Data{
		RoleID: d.RoleID,
	}
	v.src = d
	v.copyLock = rw
	return v
}

func (d *Data) IsCopy() bool {
	return d.src != nil
}

// GetBrief GetBrief Data.Queue
func (d *Data) GetBrief() *Brief {
	if !d.IsCopy() {
		return d.Brief
	}

	// 使用时拷贝
	if d.Brief == nil {
		d.Brief = &Brief{}
		if d.src.Brief != nil {
			d.copyLock.Lock()
			d.Brief = d.src.Brief.Clone()
			d.copyLock.Unlock()
		}
	}
	return d.Brief
}

// GetQueue GetQueue，请勿直接使用 Data.Queue
func (d *Data) GetQueue() map[int32][]*GenerateQueue {
	if !d.IsCopy() {
		return d.Queue
	}

	// 使用时拷贝
	if d.Queue == nil {
		d.Queue = make(map[int32][]*GenerateQueue)
		if d.src.Queue != nil {
			d.copyLock.Lock()
			for k, v2 := range d.src.Queue {
				d.Queue[k] = slices.Clone(v2)
			}
			d.copyLock.Unlock()
		}
	}
	return d.Queue
}
