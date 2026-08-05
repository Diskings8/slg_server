package game_declarations

// PollerI 角色数据 poller 接口 — 测试可注入 mock。
//
// 泛型化以避免引用 game_entitys/game_roles 造成导入环（game_roles.GetRole 返回 Poller[*Role]）。
type PollerI[T any] interface {
	Release()
	GetCopy() T
	Get() (T, error)
	Save() // 打脏标记，异步保存器持久化
}
