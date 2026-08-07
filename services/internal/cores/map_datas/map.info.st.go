package map_datas

import (
	"sync"
	"time"

	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_datas/map_buildings"
	"server.slg.com/services/internal/cores/map_datas/map_events"
)

// MapInfo 地图格子信息，包含格子的坐标、等级、类型、归属服务器以及叠加的建筑和事件
type MapInfo struct {
	rwLock           sync.RWMutex
	marchLocker      sync.Mutex
	mapID            cores_declarations.MapID
	coreMapID        cores_declarations.MapID
	configID         uint32
	Level            cores_declarations.MapLevel
	ElementType      cores_declarations.ElementType
	x                int
	y                int
	serverID         uint32
	ownerID          uint64
	isDeveloped      bool
	protectedEndTime int64
	overlayEvent     *map_events.OverlayEvent
	overlayBuilding  *map_buildings.OverlayBuilding
}

func (mi *MapInfo) GetMapID() cores_declarations.MapID {
	return mi.mapID
}

func (mi *MapInfo) GetBaseMapID() cores_declarations.MapID {
	return mi.coreMapID
}

func (mi *MapInfo) GetPointX() int {
	return mi.x
}

func (mi *MapInfo) GetPointY() int {
	return mi.y
}

func (mi *MapInfo) GetServerID() uint32 {
	return mi.serverID
}

func (mi *MapInfo) GetOwnerID() uint64 {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ownerID
}

func (mi *MapInfo) GetOverlayBuilding() *map_buildings.OverlayBuilding {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.overlayBuilding
}

func (mi *MapInfo) GetLevel() cores_declarations.MapLevel {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.Level
}

func (mi *MapInfo) GetElementID() uint32 {
	return mi.configID
}

func (mi *MapInfo) GetElementType() cores_declarations.ElementType {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.ElementType
}

// SetElement 设置地块元素类型（地形/资源/建筑），供地图初始化时调用
func (mi *MapInfo) SetElement(elementType cores_declarations.ElementType, configID uint32, level cores_declarations.MapLevel) {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.ElementType = elementType
	mi.configID = configID
	mi.Level = level
}

// GetIsDeveloped 地块是否已开发
func (mi *MapInfo) GetIsDeveloped() bool {
	mi.rwLock.RLock()
	defer mi.rwLock.RUnlock()
	return mi.isDeveloped
}

// MarkDeveloped 标记地块为已开发（开发行军胜利后调用）
func (mi *MapInfo) MarkDeveloped() {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.isDeveloped = true
}

// AddLevel 地块等级增加 inc（开发升级用），返回升级后的等级
func (mi *MapInfo) AddLevel(inc cores_declarations.MapLevel) cores_declarations.MapLevel {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.Level += inc
	return mi.Level
}

//----------------Lock----------------//

func (mi *MapInfo) TryLock() bool {
	return mi.rwLock.TryLock()
}

func (mi *MapInfo) UnLock() {
	mi.rwLock.Unlock()
}

func (mi *MapInfo) Lock() {
	mi.rwLock.Lock()
}

// LockMarchDo 行军处理锁定
func (mi *MapInfo) LockMarchDo() bool {
	if mi == nil {
		return true
	}
	return mi.marchLocker.TryLock()
}

// UnlockMarchDo 行军处理解锁
func (mi *MapInfo) UnlockMarchDo() {
	if mi == nil {
		return
	}
	mi.marchLocker.Unlock()
}

// -------------------

// Occupy 设置地块占领者
// 注意：调用方需已持有 mi 的写锁
func (mi *MapInfo) Occupy(ownerID uint64) {
	mi.ownerID = ownerID
}

// Free 地块被释放
func (mi *MapInfo) Free(now time.Time) {
	mi.rwLock.Lock()
	defer mi.rwLock.Unlock()
	mi.freeLocked(now)
}

// freeLocked 地块被释放（调用方需已持有 mi 写锁）
//
// 供 SetRoleMainCity 等已持锁场景使用，避免对 GetFreeBorn 返回的已加锁格子
// 重复 Lock 造成重入死锁（sync.RWMutex 不可重入）。
func (mi *MapInfo) freeLocked(now time.Time) {
	mi.ownerID = 0
	mi.isDeveloped = false
	if mi.ElementType != cores_declarations.ElementType_Terrain_3 {
		mi.protectedEndTime = now.Add(time.Hour).Unix()
	}
	mi.overlayEvent.AfterFree(now)
	mi.overlayBuilding.AfterFree(now)
}
