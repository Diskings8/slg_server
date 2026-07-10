package map_managers

import (
	"sync"

	"server.slg.com/api/protocol/pb/pb_camera"
)

var mapPBPool = &sync.Pool{
	New: func() any { return nil },
}

// MapPBGet 池子获取
func MapPBGet() *pb_camera.MapInfo {
	d := mapPBPool.Get()
	if d != nil {
		d.(*pb_camera.MapInfo).Reset()
		return d.(*pb_camera.MapInfo)
	}
	return &pb_camera.MapInfo{}
}

// MapPBPut 放回池子
func MapPBPut(l ...*pb_camera.MapInfo) {
	for _, v := range l {
		mapPBPool.Put(v)
	}
}
