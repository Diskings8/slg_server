package declaration_cores

type MarchID uint64

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
