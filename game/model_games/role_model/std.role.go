package role_model

type Role struct {
	Id uint64
}

func (r *Role) ID() uint64 {
	return r.Id
}
