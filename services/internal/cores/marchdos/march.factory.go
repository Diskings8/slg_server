package marchdos

import (
	"server.slg.com/services/internal/cores/cores_declarations"
	"server.slg.com/services/internal/cores/map_managers"
	"server.slg.com/services/internal/cores/marchs"
)

// marchFactories 行军类型构造函数注册表
// 各子包在 init() 中调用 RegisterMarchFactory 注册自己
var marchFactories = func() map[cores_declarations.MarchType]func(*map_managers.MapManager, *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	return make(map[cores_declarations.MarchType]func(*map_managers.MapManager, *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI)
}()

// RegisterMarchFactory 注册行军类型构造函数
//
// 由各子包（marchdos/attack/、marchdos/assist/ 等）在 init() 中调用。
// 注册后即可通过 NewMarchDo 根据 MarchType 创建对应的行军执行器。
func RegisterMarchFactory(mt cores_declarations.MarchType, factory func(*map_managers.MapManager, *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI) {
	if _, ok := marchFactories[mt]; ok {
		panic("marchdos: 重复注册行军类型 " + string(rune(mt)))
	}
	marchFactories[mt] = factory
}

// NewMarchDo 根据 MarchType 创建对应的行军执行器
//
// 通过注册表查找并调用对应的构造函数。
// 未注册的类型返回 nil。
func NewMarchDo(mm *map_managers.MapManager, marchInfo *marchs.MarchInfo) cores_declarations.MarchDoFuncHandleI {
	if marchInfo == nil || mm == nil {
		return nil
	}
	if f, ok := marchFactories[marchInfo.MarchType]; ok {
		return f(mm, marchInfo)
	}
	return nil
}
