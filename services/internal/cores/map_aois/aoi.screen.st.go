package map_aois

import (
	"sync/atomic"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.AoiScreenI = new(Screen[cores_declarations.ScreenID])

type Screen[T cores_declarations.ScreenID] struct {
	ID              T
	connect         hashmaps.Map[uint64, cores_declarations.MapRoleConnectI]
	around          *atomic.Pointer[[]*Screen[T]]
	allMarch        hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
	allPassingMarch hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
	notFreeData     hashmaps.Map[cores_declarations.MapID, struct{}] // 非空地地块集合（建筑/怪物等）
}

// MarchAdd 添加行军
func (s *Screen[T]) MarchAdd(info cores_declarations.MarchInfoI) {
	if s == nil {
		return
	}
	s.allMarch.Store(info.GetMarchID(), info)
	info.AddAOIBlock(s)
}

// MarchDelete 删除行军
func (s *Screen[T]) MarchDelete(info cores_declarations.MarchInfoI) {
	if s == nil {
		return
	}
	s.allMarch.Delete(info.GetMarchID())
}

// MarchRange 行军Range
func (s *Screen[T]) MarchRange(f func(info cores_declarations.MarchInfoI) bool) {
	if s == nil {
		return
	}
	s.allMarch.Range(func(_ cores_declarations.MarchID, value cores_declarations.MarchInfoI) bool {
		return f(value)
	})
}

// PassingMarchAdd 添加路过行军
func (s *Screen[T]) PassingMarchAdd(info cores_declarations.MarchInfoI) {
	if s == nil {
		return
	}
	s.allPassingMarch.Store(info.GetMarchID(), info)
	info.AddPassingAOIBlock(s)
}

// PassingMarchDelete 删除路过行军
func (s *Screen[T]) PassingMarchDelete(info cores_declarations.MarchInfoI) {
	if s == nil {
		return
	}
	s.allPassingMarch.Delete(info.GetMarchID())
}

// PassingMarchRange 路过行军Range
func (s *Screen[T]) PassingMarchRange(f func(info cores_declarations.MarchInfoI) bool) {
	if s == nil {
		return
	}
	s.allPassingMarch.Range(func(_ cores_declarations.MarchID, value cores_declarations.MarchInfoI) bool {
		return f(value)
	})
}

// Connects 在视野内的角色连接
func (s *Screen[T]) Connects(connects map[uint64]cores_declarations.MapRoleConnectI) map[uint64]cores_declarations.MapRoleConnectI {
	if s == nil {
		return connects
	}
	s.connect.Range(func(k uint64, v cores_declarations.MapRoleConnectI) bool {
		connects[v.GetRoleID()] = v
		return true
	})
	return connects
}

// ConnectRoleIDs 获取视野中的链接的玩家id
func (s *Screen[T]) ConnectRoleIDs(connects *[]uint64) *[]uint64 {
	if s == nil {
		return connects
	}
	s.connect.Range(func(_ uint64, conn cores_declarations.MapRoleConnectI) bool {
		*connects = append(*connects, conn.GetRoleID())
		return true
	})
	return connects
}

// MapDataAdd 添加非空地地块记录
func (s *Screen[T]) MapDataAdd(mapID cores_declarations.MapID) {
	if s == nil {
		return
	}
	s.notFreeData.Store(mapID, struct{}{})
}

// MapDataDel 移除非空地地块记录
func (s *Screen[T]) MapDataDel(mapID cores_declarations.MapID) {
	if s == nil {
		return
	}
	s.notFreeData.Delete(mapID)
}

// MapDataRange 遍历非空地地块
func (s *Screen[T]) MapDataRange(f func(mapID cores_declarations.MapID) bool) {
	if s == nil {
		return
	}
	s.notFreeData.Range(func(id cores_declarations.MapID, _ struct{}) bool {
		return f(id)
	})
}
