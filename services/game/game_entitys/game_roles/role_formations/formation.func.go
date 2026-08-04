package role_formations

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_maps_march"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewRoleFormations(roleID uint64) *RoleFormations {
	return &RoleFormations{
		RoleID: roleID,
		List:   make([]*game_models.RoleFormation, 0),
	}
}

func (rfs *RoleFormations) Init() {
	for _, modelOne := range rfs.List {
		formation := NewRoleFormation(modelOne)
		rfs.Mem.Store(modelOne.ID, formation)
	}
}

func (rfs *RoleFormations) Copy(src *RoleFormations) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, rfs)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	rfs.Init()
}

//-------------------------------

func NewRoleFormation(one *game_models.RoleFormation) *RoleFormation {
	return &RoleFormation{
		RoleFormation: one,
	}
}

// GetFormationByID 按唯一队列ID获取队伍
func (rfs *RoleFormations) GetFormationByID(id uint64) *RoleFormation {
	if v, ok := rfs.Mem.Load(id); ok {
		return v
	}
	return nil
}

// FormationHasHero 是否存在任何队列引用了该英雄（用于升星消耗等防误删）
func (rfs *RoleFormations) FormationHasHero(heroID uint64) bool {
	for _, f := range rfs.List {
		for _, hs := range f.HeroSlots {
			if hs.GetHeroId() == heroID {
				return true
			}
		}
	}
	return false
}

// CreateFormation 分配一个队列（校场等级/建筑完成时调用），返回新队列
func (rfs *RoleFormations) CreateFormation(roleID, cityID uint64) *RoleFormation {
	formation := &game_models.RoleFormation{
		RoleID:    roleID,
		CityID:    cityID,
		HeroSlots: make([]*pb_maps_march.HeroSlot, 0),
	}
	formation.ID = snowflakes.GenUUID()
	rfs.AddFormation(formation)
	return NewRoleFormation(formation)
}

// ListByCity 获取队列列表；cityID=0 返回全部建筑队列
func (rfs *RoleFormations) ListByCity(cityID uint64) []*game_models.RoleFormation {
	list := make([]*game_models.RoleFormation, 0)
	for _, modelOne := range rfs.List {
		if cityID == 0 || modelOne.CityID == cityID {
			list = append(list, modelOne)
		}
	}
	return list
}

// AddFormation 新增/更新编队（按ID去重）
func (rfs *RoleFormations) AddFormation(formation *game_models.RoleFormation) {
	for i, v := range rfs.List {
		if v.ID == formation.ID {
			rfs.List[i] = formation
			rfs.Init()
			return
		}
	}
	rfs.List = append(rfs.List, formation)
	rfs.Init()
}

// DeleteCityFormations 删除某城市全部队列（城市拆除时）
func (rfs *RoleFormations) DeleteCityFormations(cityID uint64) {
	kept := rfs.List[:0]
	for _, v := range rfs.List {
		if v.CityID != cityID {
			kept = append(kept, v)
		}
	}
	rfs.List = kept
	rfs.Init()
}

// DeleteFormation 删除单个队列
func (rfs *RoleFormations) DeleteFormation(id uint64) {
	for i, v := range rfs.List {
		if v.ID == id {
			rfs.List = append(rfs.List[:i], rfs.List[i+1:]...)
			rfs.Init()
			return
		}
	}
}
