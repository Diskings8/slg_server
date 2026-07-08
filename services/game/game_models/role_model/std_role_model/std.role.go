package std_role_model

import (
	"sync/atomic"

	"server.slg.com/common/pollers"
)

var _ pollers.IHandler = (*Role)(nil)

// Role 角色逻辑模型，实现 IHandler 接口，提供缓存和数据库持久化能力
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
