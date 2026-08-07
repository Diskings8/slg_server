package role_buildings

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

// BuildingCostI 建筑建造/升级的资源消耗接口（预留）
//
// 货币类道具引入后实现具体消耗逻辑（CheckCost/DeductCost）。
// TODO: 接入货币道具后提供默认实现。
type BuildingCostI interface {
	// CheckCost 检查资源是否足够（预留）
	CheckCost(role common_declarations.DataI) error
	// DeductCost 扣除资源（预留）
	DeductCost(role common_declarations.DataI) error
}

func NewRoleBuildings(roleID uint64) *RoleBuildings {
	return &RoleBuildings{
		RoleID: roleID,
		List:   make([]*game_models.RoleBuilding, 0),
	}
}

func (rbs *RoleBuildings) Init() {
	for _, modelOne := range rbs.List {
		building := NewRoleBuilding(modelOne)
		rbs.Mem.Store(modelOne.ID, building)
	}
}

func (rbs *RoleBuildings) Copy(src *RoleBuildings) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, rbs)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	rbs.Init()
}

//-------------------------------

func NewRoleBuilding(one *game_models.RoleBuilding) *RoleBuilding {
	return &RoleBuilding{
		RoleBuilding: one,
	}
}

// GetBuilding 按建筑ID获取
func (rbs *RoleBuildings) GetBuilding(id uint64) *RoleBuilding {
	if v, ok := rbs.Mem.Load(id); ok {
		return v
	}
	return nil
}

// GetMainCity 获取主城
func (rbs *RoleBuildings) GetMainCity() *RoleBuilding {
	for _, modelOne := range rbs.List {
		if modelOne.Type == pb_city.BuildingType_RoleMainCity {
			return NewRoleBuilding(modelOne)
		}
	}
	return nil
}

// GetBarrackByCity 获取归属指定城市且已完成的兵营（兵力上限加成）
func (rbs *RoleBuildings) GetBarrackByCity(cityID uint64) *RoleBuilding {
	if cityID == 0 {
		return nil
	}
	for _, modelOne := range rbs.List {
		if modelOne.Type == pb_city.BuildingType_RoleBarracks &&
			modelOne.CityID == cityID &&
			modelOne.State == pb_city.BuildingState_Completed {
			return NewRoleBuilding(modelOne)
		}
	}
	return nil
}

// HasMilitaryBuilding 是否已建造完成的军事建筑（解锁上阵队伍）
func (rbs *RoleBuildings) HasMilitaryBuilding() bool {
	for _, modelOne := range rbs.List {
		if modelOne.Type == pb_city.BuildingType_RoleMilitary && modelOne.State == pb_city.BuildingState_Completed {
			return true
		}
	}
	return false
}

// AddBuilding 添加建筑
func (rbs *RoleBuildings) AddBuilding(building *game_models.RoleBuilding) {
	rbs.List = append(rbs.List, building)
	rbs.Init()
}

// DeleteBuilding 删除建筑
func (rbs *RoleBuildings) DeleteBuilding(id uint64) {
	for i, v := range rbs.List {
		if v.ID == id {
			rbs.List = append(rbs.List[:i], rbs.List[i+1:]...)
			rbs.Init()
			return
		}
	}
}
