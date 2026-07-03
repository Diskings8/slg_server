package aois

import (
	"server.slg.com/services/internal/cores/cores_declarations"
)

type Screen[T int32 | uint32] struct {
}

func (i Screen[T]) MarchAdd(info cores_declarations.MarchInfoI) {
	panic("implement me")
}

func (i Screen[T]) CrossMarchAdd(info cores_declarations.MarchInfoI) {
	panic("implement me")
}
