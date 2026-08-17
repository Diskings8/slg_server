package role_buildings

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_city"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

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

// GetBarrackByCity 获取归属指定城市的兵营（兵力上限加成）
// 注意：不过滤 State==Completed——升级中兵营旧等级加成继续生效；新建 Level=0 贡献 0 天然正确。
func (rbs *RoleBuildings) GetBarrackByCity(cityID uint64) *RoleBuilding {
	if cityID == 0 {
		return nil
	}
	for _, modelOne := range rbs.List {
		if modelOne.Type == pb_city.BuildingType_RoleBarracks && modelOne.CityID == cityID {
			return NewRoleBuilding(modelOne)
		}
	}
	return nil
}

// GetDrillByCity 获取归属指定城市的校场
func (rbs *RoleBuildings) GetDrillByCity(cityID uint64) *RoleBuilding {
	if cityID == 0 {
		return nil
	}
	for _, modelOne := range rbs.List {
		if modelOne.Type == pb_city.BuildingType_RoleDrill && modelOne.CityID == cityID {
			return NewRoleBuilding(modelOne)
		}
	}
	return nil
}

// BuildingExists 校验建筑是否已存在：
// 城市类（主城/分城）不限数量；附属类（校场/兵营/城墙/仓库/资源）同 city 同类型唯一。
func (rbs *RoleBuildings) BuildingExists(t pb_city.BuildingType, cityID uint64) bool {
	for _, modelOne := range rbs.List {
		if modelOne.Type != t {
			continue
		}
		// 城市类：不限制数量（主城/分城可多个）
		if t == pb_city.BuildingType_RoleMainCity || t == pb_city.BuildingType_RoleBranchCity {
			continue
		}
		// 附属类：同 city 同类型唯一
		if modelOne.CityID == cityID {
			return true
		}
	}
	return false
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
