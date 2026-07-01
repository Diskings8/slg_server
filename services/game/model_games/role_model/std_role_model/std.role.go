package std_role_model

import (
	"sync/atomic"

	"server.slg.com/common/pools"
)

var _ pools.IHandler = (*Role)(nil)

type Role struct {
	Id    uint64
	dirty atomic.Bool
}

func (r *Role) ID() uint64 {
	return r.Id
}

func (r *Role) Dirty() {
	r.dirty.Swap(true)
}

func (r *Role) IsDirty() bool {
	return r.dirty.Load()
}

func (r *Role) SaveCache() error {
	//TODO implement me
	panic("implement me")
}

func (r *Role) SaveDB() error {
	//TODO implement me
	panic("implement me")
}
