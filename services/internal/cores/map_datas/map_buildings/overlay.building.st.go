package map_buildings

import (
	"sync"
	"time"

	"server.slg.com/services/internal/cores/cores_declarations"
)

// OverlayBuilding 地图建筑覆盖层数据，表示地图格子上叠加的建筑信息
// 永久不可变的是土地类型
// 类角色相关的都统属于建筑
type OverlayBuilding struct {
	sync.RWMutex
	building cores_declarations.BuildingI
}

func (ob *OverlayBuilding) GetBuilding() cores_declarations.BuildingI {
	return ob.building
}

func (ob *OverlayBuilding) AfterFree(now time.Time) {

}
