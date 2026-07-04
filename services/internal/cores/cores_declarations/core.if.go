package cores_declarations

import "server.slg.com/api/protocol/pb/pb_camera"

type AoiScreenI interface {
}

type MarchHero interface {
}

type MarchSoldier interface {
	GetCurCount() uint64
	GetMaxCount() uint64
	GetInjuredCount() uint64
}

type MarchInfoI interface {
	GetMarchID() MarchID
	AddPassingAOIBlock(AoiScreenI)
	AddAOIBlock(AoiScreenI)
}

// MarchDoFuncHandleI 行军处理接口
type MarchDoFuncHandleI interface {
	Do() error
	LockDo(marchLock, fromMapLock, toMapLock bool) error
	CallBack() error
	CallBackNow() error
	Lock(marchDoLock, fromMapLock, toMapLock bool) bool
	Unlock()
	Leave() error
}

// MapRoleConnect 地图服务上的角色连接
type MapRoleConnect interface {
	// GetOldScreenMapID 取得上次城外屏幕中心点
	GetOldScreenMapID() MapID
	// GetRoleID 取得角色ID
	GetRoleID() uint64
	// GetScaleLevel 取得显示等级
	GetScaleLevel() pb_camera.CameraLayer
	// GetScreenMapID 地图ID
	GetScreenMapID() MapID
}

type MapConfigI interface {
	// MapCount 地图总数
	MapCount() uint32

	MapID2XY(id MapID) (x, y int32)

	XY2MapID(x int32, y int32) MapID

	// SortByDis 距离排序
	SortByDis(mapID MapID, mapIDs []MapID)
}
