package cores_declarations

type MarchID uint64

type MapID int32

const (
	InvalidMapID = -1
)

type MarchTimeType int

const (
	MarchTimeType_Straight MarchTimeType = iota
)

type MarchType uint32

const (
	MarchType_110101 MarchType = 110101
)

type MarchState uint32

const (
	MarchState_Idle MarchState = iota
)

const (
	HeroPose_0 = iota
	HeroPose_1
	HeroPose_2
)

type MapGroup uint32

const (
	MapGroupBase MapGroup = iota
)

type RoleMainCityState int

const (
	RoleMainCityStateNormal RoleMainCityState = iota
	RoleMainCityStatePortable
)

const (
	RoleMainCityStateNormalCoverCount   = 9
	RoleMainCityStatePortableCoverCount = 1
)

const (
	// Land1CoverBaseKey 1*1主位置在那一个键里。
	Land1CoverBaseKey = 1
	// Land3CoverBaseKey 3*3主位置在那一个键里。
	Land3CoverBaseKey = 4
)
