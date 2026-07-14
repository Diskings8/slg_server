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
	around          *atomic.Pointer[[]*Screen[T]] // 周围一圈的切块
	allMarch        hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
	allPassingMarch hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
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
// connects 参数用于优化gc
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
// connects 参数用于优化gc
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
