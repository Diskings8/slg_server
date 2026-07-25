package game_roles

func Get() *Role {
	return rolePool.Get()
}

// Release 释放到对象池中
func Release(r *Role) {
	rolePool.Put(r)
}
