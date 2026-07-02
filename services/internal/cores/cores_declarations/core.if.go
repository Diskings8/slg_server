package cores_declarations

type AoiScreen interface {
}

type MarchHero interface {
}

type MarchSoldier interface {
	GetCurCount() uint64
	GetMaxCount() uint64
	GetInjuredCount() uint64
}
