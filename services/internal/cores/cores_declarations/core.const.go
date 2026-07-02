package cores_declarations

type MarchID uint64

type MapID uint32

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
