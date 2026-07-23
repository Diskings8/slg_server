package cores_declarations

type MarchID uint64

func (i MarchID) Uint64() uint64 {
	return uint64(i)
}

type MapID int32

func (i MapID) Int32() int32 {
	return int32(i)
}

func (i MapID) IsInvalid() bool {
	return i == InvalidMapID
}

const (
	InvalidMapID = -1
)

type ScreenID int32

func (i ScreenID) Int32() int32 {
	return int32(i)
}

type BornBlockID int32

// MarchTimeType 行军耗时类型
type MarchTimeType int

const (
	MarchTimeTypeStraight MarchTimeType = iota
)

// MarchType 行军类型
type MarchType uint32

const (
	MarchTypeAttack   MarchType = 10001 // 攻击
	MarchTypeAssist   MarchType = 10002 // 驻守
	MarchTypeSweep    MarchType = 10003 // 扫荡
	MarchTypeStrategy MarchType = 10004 // 计略
	MarchTypeDevelop  MarchType = 10005 // 开发
)

// 布阵槽位
const (
	TeamSlot1 = iota + 1
	TeamSlot2
	TeamSlot3
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

// ScreenWeight 屏幕宽度
const ScreenWeight = 40

const (
	RoleMainCityStateNormalCoverCount   = 9
	RoleMainCityStatePortableCoverCount = 1
)

const (
	// HallLandCover 玩家城边长
	HallLandCover = 3
	// HallCoverCount 玩家城占地位置数量
	HallCoverCount = 9
	// Land1CoverBaseKey 1*1主位置在那一个键里。
	Land1CoverBaseKey = 1
	// Land3CoverBaseKey 3*3主位置在那一个键里。
	Land3CoverBaseKey = 4
	// Land5CoverBaseKey 5*5主位置在那一个键里。
	Land5CoverBaseKey = 12
	// Land7CoverBaseKey 7*7主位置在那一个键里。
	Land7CoverBaseKey = 24
	// Land11CoverBaseKey 11*11主位置在那一个键里。
	Land11CoverBaseKey = 60
)

const (
	// ServerMapBlockCutNum 本服地图切块数量
	ServerMapBlockCutNum = 25
	// ServerMapBlockRowCutNum 本服地图每行切块数量
	ServerMapBlockRowCutNum = 5
)

type ScaleLevel int

const (
	ScaleLevel0 ScaleLevel = iota
	ScaleLevel1
	ScaleLevel2
	ScaleLevel3
	ScaleLevel4
	ScaleLevel5
)

type MapLevel int

// ElementType 地块元素类型
type ElementType int

func (i ElementType) IsCantBornUse() bool {
	return i != ElementType_Terrain_1 &&
		i != ElementType_Terrain_2 &&
		i != ElementType_Terrain_3
}

const (
	ElementType_None        ElementType = iota
	ElementType_Resources_1             // 资源1
	ElementType_Resources_2             // 资源2
	ElementType_Resources_3             // 资源3
	ElementType_Resources_4             // 资源4
	ElementType_Terrain_1               // 地形1--山
	ElementType_Terrain_2               // 地形2--水
	ElementType_Terrain_3               // 地形3--战乱地
)
