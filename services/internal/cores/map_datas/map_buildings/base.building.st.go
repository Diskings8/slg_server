package map_buildings

import (
	"sync"
	"time"

	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/services/internal/cores/cores_declarations"
)

var _ cores_declarations.BuildingI = new(BaseBuilding)

type BaseBuilding struct {
	BuildingsType          pb_city.BuildingType // 建筑类型
	BuildingsMaxHp         uint64               // 最大血量
	BuildingsCurHp         uint64               // 当前血量
	BuildingsConfID        uint32               // 建筑配置ID
	BuildingsLevel         uint32               // 当前等级
	BuildingsRecoverHpTime int64                // 上次恢复hp血量时间
	BuildingsConf          cores_declarations.BaseBuildingConfI
	buildingsRWLock        sync.RWMutex
}

func NewBaseBuilding(confID, curLv uint32, conf cores_declarations.BaseBuildingConfI) *BaseBuilding {
	maxHp := conf.GetBuildingsMaxHp(confID, curLv)
	buildings := &BaseBuilding{
		BuildingsConfID:        confID,
		BuildingsCurHp:         maxHp,
		BuildingsLevel:         curLv,
		BuildingsRecoverHpTime: 0,
		BuildingsConf:          conf,
	}
	return buildings
}

func (b *BaseBuilding) GetBuildingType() pb_city.BuildingType {
	return b.BuildingsType
}

func (b *BaseBuilding) AfterFree(time.Time) {

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
