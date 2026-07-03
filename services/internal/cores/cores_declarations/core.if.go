package cores_declarations

type AoiScreenI interface {
}

type MarchHero interface {
}

type MarchSoldier interface {
	GetCurCount() uint64
	GetMaxCount() uint64
	GetInjuredCount() uint64
}

type MarchInfoI interface {
	GetMarchID() MarchID
	AddPassingAOIBlock(AoiScreenI)
	AddAOIBlock(AoiScreenI)
}

// MarchDoFuncHandleI 行军处理接口
type MarchDoFuncHandleI interface {
	Do() error
	LockDo(marchLock, fromMapLock, toMapLock bool) error
	CallBack() error
	CallBackNow() error
	Lock(marchDoLock, fromMapLock, toMapLock bool) bool
	Unlock()
	Leave() error
}
