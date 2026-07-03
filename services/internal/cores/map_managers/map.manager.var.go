package map_managers

import "sync"

var mapPBPool = &sync.Pool{
	New: func() any { return nil },
}

// MapPBGet todo
func MapPBGet() any {
	//d := mapPBPool.Get()
	//if d != nil {
	//	d.(*pb_wmcamera.MapInfo).Reset()
	//	return d.(*pb_wmcamera.MapInfo)
	//}
	//return &pb_wmcamera.MapInfo{}
	return nil
}

// MapPBPut todo
func MapPBPut(l ...any) {
	for _, v := range l {
		mapPBPool.Put(v)
	}
}
