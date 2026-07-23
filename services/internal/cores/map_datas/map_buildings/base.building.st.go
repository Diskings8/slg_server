package map_buildings

import (
	"sync"
	"time"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.BuildingI = new(BaseBuilding)

type BaseBuilding struct {
	ID                     uint64               // 建筑id
	BuildingsType          pb_city.BuildingType // 建筑类型
	BuildingsMaxHp         uint64               // 最大血量
	BuildingsCurHp         uint64               // 当前血量
	BuildingsConfID        uint32               // 建筑配置ID
	BuildingsLevel         uint32               // 当前等级
	BuildingsRecoverHpTime int64                // 上次恢复hp血量时间
	BuildingsConf          cores_declarations.BaseBuildingConfI
	VisionRangeVal         int32 // 战争视野范围（格），建造时从配置写入
	buildingsRWLock        sync.RWMutex

	aoiReg cores_declarations.AOIRegistrar // 用于建造/放弃时更新 AOI
	mapID  cores_declarations.MapID        // 自身所在的地图格子

	ConstructionEndTime int64 // 建造完成时间戳；0=未开始/未完成
	IsCompleted         bool  // 建造是否已完成
	IsAbandoning        bool  // 是否正在放弃中
	AbandonEndTime      int64 // 放弃完成时间戳
}

func NewBaseBuilding(confID, curLv uint32, conf cores_declarations.BaseBuildingConfI) *BaseBuilding {
	maxHp := conf.GetBuildingsMaxHp(confID, curLv)
	buildings := &BaseBuilding{
		BuildingsConfID:        confID,
		BuildingsCurHp:         maxHp,
		BuildingsLevel:         curLv,
		BuildingsRecoverHpTime: 0,
		BuildingsConf:          conf,
		VisionRangeVal:         2, // 建筑默认 2 格战争视野
	}
	return buildings
}

// BindMap 绑定地图位置，用于建造/放弃时更新 AOI
func (b *BaseBuilding) BindMap(aoiReg cores_declarations.AOIRegistrar, mapID cores_declarations.MapID) {
	b.aoiReg = aoiReg
	b.mapID = mapID
}

func (b *BaseBuilding) GetBuildingType() pb_city.BuildingType {
	return b.BuildingsType
}

func (b *BaseBuilding) AfterFree(time.Time) {

}

func (b *BaseBuilding) VisionRange() int32 {
	if !b.IsCompleted {
		return 0 // 未完成不提供建筑视野
	}
	return b.VisionRangeVal
}

// IsUnderConstruction 是否正在建造中
func (b *BaseBuilding) IsUnderConstruction() bool {
	if b.IsCompleted {
		return false
	}
	return b.ConstructionEndTime > 0
}

// StartConstruction 开始建造
func (b *BaseBuilding) StartConstruction(endTime int64) {
	b.buildingsRWLock.Lock()
	defer b.buildingsRWLock.Unlock()
	b.ConstructionEndTime = endTime
	b.IsCompleted = false
	b.IsAbandoning = false
	b.AbandonEndTime = 0
}

// CompleteConstruction 建造完成，注册到 AOI
func (b *BaseBuilding) CompleteConstruction() {
	b.buildingsRWLock.Lock()
	b.IsCompleted = true
	b.ConstructionEndTime = 0
	aoiReg := b.aoiReg
	mapID := b.mapID
	b.buildingsRWLock.Unlock()

	if aoiReg != nil && mapID >= 0 {
		aoiReg.MapDataAdd(mapID)
	}
}

// StartAbandon 开始放弃
func (b *BaseBuilding) StartAbandon(endTime int64) {
	b.buildingsRWLock.Lock()
	defer b.buildingsRWLock.Unlock()
	b.IsAbandoning = true
	b.AbandonEndTime = endTime
}

// CompleteAbandon 放弃完成，从 AOI 注销
func (b *BaseBuilding) CompleteAbandon() {
	b.buildingsRWLock.Lock()
	b.IsCompleted = false
	b.IsAbandoning = false
	b.ConstructionEndTime = 0
	b.AbandonEndTime = 0
	b.BuildingsLevel = 0
	aoiReg := b.aoiReg
	mapID := b.mapID
	b.buildingsRWLock.Unlock()

	if aoiReg != nil && mapID >= 0 {
		aoiReg.MapDataDel(mapID)
	}
}

func (b *BaseBuilding) BeforeBeAttack(cores_declarations.MarchInfoI) bool {
	return true
}

func (b *BaseBuilding) LevelUp() {
	b.buildingsRWLock.Lock()
	defer b.buildingsRWLock.Unlock()
	if b.BuildingsLevel < b.BuildingsConf.GetBuildingsMaxLevel() {
		b.BuildingsLevel++
	}
	b.BuildingsRecoverHpTime = time.Now().Unix()
}

// AddBuildingsHp 增加建筑血量
//
//	ip: add 增加的数值
//	op: right  生效增加的数值
func (b *BaseBuilding) AddBuildingsHp(add uint64) (right uint64) {
	b.buildingsRWLock.Lock()
	defer b.buildingsRWLock.Unlock()
	maxHp := b.BuildingsConf.GetBuildingsMaxHp(b.BuildingsConfID, b.BuildingsLevel)
	if b.BuildingsCurHp < maxHp {
		if maxHp-b.BuildingsCurHp < add {
			right = maxHp - b.BuildingsCurHp
			b.BuildingsCurHp = maxHp
		} else {
			b.BuildingsCurHp += add
			right = add
		}
	}
	return
}

// ReduceBuildingsHp 减少建筑血量
//
//	ip: reduce 减少的数值
//	op: right  生效减少的数值
//	op: isBroken 是否以及损毁
func (b *BaseBuilding) ReduceBuildingsHp(reduce uint64) (right uint64, isBroken bool) {
	b.buildingsRWLock.Lock()
	defer b.buildingsRWLock.Unlock()
	if b.BuildingsCurHp < reduce {
		b.BuildingsCurHp = 0
		isBroken = true
	} else {
		b.BuildingsCurHp -= reduce
		isBroken = false
	}
	return
}

func (b *BaseBuilding) BeAttack(info cores_declarations.MarchInfoI) (right uint64, isBroken bool) {
	return b.ReduceBuildingsHp(info.GetRelocationVal())
}
