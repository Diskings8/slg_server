package cores_declarations

import (
	"time"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/api/protocol/pb/pb_common"
)

type AoiScreenI interface {
	MarchDelete(info MarchInfoI)
	PassingMarchDelete(info MarchInfoI)
}

type MarchHeroI interface {
}

type MarchSoldierI interface {
	GetCurCount() uint64
	GetMaxCount() uint64
	GetInjuredCount() uint64
}

type MarchInfoI interface {
	GetMarchID() MarchID
	GetUnionID() uint64
	AddPassingAOIBlock(AoiScreenI)
	AddAOIBlock(AoiScreenI)
	GetRelocationVal() uint64 // 获取拆迁值
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

// MapRoleConnectI 地图服务上的角色连接
type MapRoleConnectI interface {
	// GetRoleID 取得角色ID
	GetRoleID() uint64
	// GetScreenMapID 地图ID
	GetScreenMapID() MapID
	// SetScreenMapID 设置屏幕的地图ID
	SetScreenMapID(id MapID)
	//Send 发包
	Send(packet *pb_common.NodePacket) error
}

// BornBlockI 出生块接口
type BornBlockI interface {
	Store(bornID BornBlockID, data map[int32]struct{}) bool
	Load(bornID BornBlockID) (map[int32]struct{}, bool)
	Use(bornID BornBlockID) bool
	Free(bornID BornBlockID) bool
	Delete(bornID BornBlockID)
	Range(f func(BornBlockID, map[int32]struct{}) bool)
}

type MapConfigI interface {
	// MapCount 地图总数
	MapCount() int32
	// MapScope 地图每行格子数量
	MapScope() int32

	MapID2XY(id MapID) (x, y int32)

	XY2MapID(x int32, y int32) MapID

	// SortByDis 距离排序
	SortByDis(mapID MapID, mapIDs []MapID)
	// CoverMapIDs 获取覆盖到的地图id
	CoverMapIDs(id int32, i int, i2 any) []MapID
}

type BaseBuildingConfI interface {
	GetBuildingsMaxHp(buildingId uint32, buildingLv uint32) uint64
	GetBuildingsMaxLevel() uint32
}

type NpcBuildingConfI interface {
	BaseBuildingConfI
}

// AOIRegistrar AOI 地块注册接口，用于建筑建造/放弃时更新 AOI 记录
type AOIRegistrar interface {
	MapDataAdd(mapID MapID)
	MapDataDel(mapID MapID)
}

type BuildingI interface {
	GetBuildingType() pb_city.BuildingType
	AfterFree(freeTime time.Time)
	BeforeBeAttack(info MarchInfoI) bool
	BeAttack(info MarchInfoI) (right uint64, isBroken bool)
	LevelUp()
	VisionRange() int32 // 战争视野范围（格），0 表示仅自身地块
}
