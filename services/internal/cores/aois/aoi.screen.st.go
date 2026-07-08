package aois

import (
	"sync/atomic"

	"server.slg.com/common/utils/hashmaps"
	"server.slg.com/services/internal/cores/cores_declarations"
)

type Screen[T cores_declarations.ScreenID] struct {
	ID              T
	connect         hashmaps.Map[uint64, cores_declarations.MapRoleConnectI]
	around          *atomic.Pointer[[]*Screen[T]] // 周围一圈的切块
	allMarch        hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
	allPassingMarch hashmaps.Map[cores_declarations.MarchID, cores_declarations.MarchInfoI]
}

func (s *Screen[T]) MarchAdd(info cores_declarations.MarchInfoI) {
	s.allMarch.Store(info.GetMarchID(), info)
}

func (s *Screen[T]) PassingMarchAdd(info cores_declarations.MarchInfoI) {
	s.allPassingMarch.Store(info.GetMarchID(), info)
}

func (s *Screen[T]) Connects(connects map[uint64]cores_declarations.MapRoleConnectI) map[uint64]cores_declarations.MapRoleConnectI {
	panic("implement me")
}
