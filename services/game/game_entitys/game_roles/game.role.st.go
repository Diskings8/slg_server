package game_roles

import (
	"reflect"
	"strconv"
	"sync"

	"server.slg.com/api/protocol/pb/pb_gateway"
	"server.slg.com/common/common_declarations"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_entitys/game_roles/cultivate_costs"
	"server.slg.com/services/game/game_entitys/game_roles/hero_skillcollections"
	"server.slg.com/services/game/game_entitys/game_roles/hero_skills"
	"server.slg.com/services/game/game_entitys/game_roles/role_attrs"
	"server.slg.com/services/game/game_entitys/game_roles/role_buildings"
	"server.slg.com/services/game/game_entitys/game_roles/role_formations"
	"server.slg.com/services/game/game_entitys/game_roles/role_heroes"
	"server.slg.com/services/game/game_entitys/game_roles/role_items"
	"server.slg.com/services/game/game_entitys/game_roles/role_recruits"
)

var _ common_declarations.DataI = new(Role)

// Role 游戏角色
type Role struct {
	ID uint64 `json:"id,omitempty"`

	src      *Role         // 原始数据，用于获取副本数据
	copyLock *sync.RWMutex // 拷贝时使用
	isDelete bool          // 是否删除

	Status        int32                 `json:"status,omitempty"`          // 状态(1:正常,2:封号,3:禁言)
	StatusEndTime int64                 `json:"status_end_time,omitempty"` // 状态结束时间(单位:秒)
	BandStatus    int32                 `json:"band_status"`
	BandFinishUx  int64                 `json:"band_finish_ux"`
	GateID        uint32                `json:"-"`
	IP            string                `json:"-"`
	DeviceType    pb_gateway.DeviceType `json:"-"`

	// ── 子模块 ──
	Heroes           *role_heroes.RoleHeroes                     `json:"heroes,omitempty"`
	Skills           *hero_skills.HeroSkills                     `json:"skills,omitempty"`
	SkillCollections *hero_skillcollections.HeroSkillCollections `json:"skill_collections,omitempty"`
	CultivateCosts   *cultivate_costs.CultivateCosts             `json:"cultivate_costs,omitempty"`
	Items            *role_items.RoleItems                       `json:"items,omitempty"`
	Buildings        *role_buildings.RoleBuildings               `json:"buildings,omitempty"`
	Formations       *role_formations.RoleFormations             `json:"formations,omitempty"`
	Recruits         *role_recruits.RoleRecruits                 `json:"recruits,omitempty"`
	Attr             *role_attrs.RoleAttrs                       `json:"attr,omitempty"`
}

// UniqueID 唯一 id
func (r *Role) UniqueID() uint64 {
	return r.ID
}

// CacheKey 缓存键
func (r *Role) CacheKey() string {
	return strconv.FormatUint(r.ID, 10)
}

// Tag 实体类型名称，用于生成缓存 key
func (r *Role) Tag() string {
	return "game_role"
}

// TableName 数据库表名
func (r *Role) TableName() string {
	return "game_role"
}

// Save 保存到数据库
func (r *Role) Save(isDelete bool) error {
	if isDelete {
		return nil
	}
	return r.DBSave(nil)
}

// IsDelete 是否已标记为删除
func (r *Role) IsDelete() bool {
	return r.isDelete
}

// Marshal 编码，用于存入缓存
func (r *Role) Marshal() ([]byte, error) {
	return util_jsons.Marshal(r)
}

// Unmarshal 解码，从缓存中加载数据时使用
func (r *Role) Unmarshal(b []byte) error {
	r.New()
	err := util_jsons.Unmarshal(b, r)
	if err == nil {
		r.Init()
	}
	return err
}

// JSON2Bytes 获取存储的上一次编码数据
func (r *Role) JSON2Bytes() []byte {
	if b, ok := jsonCache.Get(r.CacheKey()); ok {
		return b.([]byte)
	}
	return nil
}

// Bytes2JSON 存储本次编码后的数据，用于下次比较是否有变化
func (r *Role) Bytes2JSON(b []byte) {
	if b == nil {
		jsonCache.Delete(r.CacheKey())
	} else {
		jsonCache.SetDefault(r.CacheKey(), b)
	}
}

// Init 初始化
// 遍历所有字段，对实现了 Init() 接口的子模块调用 Init
func (r *Role) Init() {
	r.GateID = 0

	val := reflect.ValueOf(r).Elem()
	for _, field := range val.Fields() {
		if !field.CanInterface() {
			continue
		}
		if module, ok := field.Interface().(interface{ Init() }); ok {
			module.Init()
		}
	}
}

// New 创建子模块数据
func (r *Role) New() {
	roleID := r.ID

	r.Heroes = role_heroes.NewRoleHeroes(roleID)
	r.Skills = hero_skills.NewHeroSkills(roleID)
	r.SkillCollections = hero_skillcollections.NewHeroSkillCollections(roleID)
	r.CultivateCosts = cultivate_costs.NewCultivateCosts(roleID)
	r.Items = role_items.NewRoleItems(roleID)
	r.Buildings = role_buildings.NewRoleBuildings(roleID)
	r.Formations = role_formations.NewRoleFormations(roleID)
	r.Recruits = role_recruits.NewRoleRecruits(roleID)
	r.Attr = role_attrs.NewRoleAttrs(roleID)
}

// SetStatus 设置状态
func (r *Role) SetStatus(status int32, statusEndTime int64) {
	r.Status = status
	r.StatusEndTime = statusEndTime
}

// ── Getter 方法（副本模式：使用时懒拷贝） ──

// GetHeroes 获取英雄模块
func (r *Role) GetHeroes() *role_heroes.RoleHeroes {
	if !r.IsCopy() {
		return r.Heroes
	}
	if r.Heroes == nil {
		r.Heroes = role_heroes.NewRoleHeroes(r.ID)
		if r.src.Heroes != nil {
			r.copyLock.RLock()
			r.Heroes.Copy(r.src.Heroes)
			r.copyLock.RUnlock()
		}
		r.Heroes.Init()
	}
	return r.Heroes
}

// GetSkills 获取技能模块
func (r *Role) GetSkills() *hero_skills.HeroSkills {
	if !r.IsCopy() {
		return r.Skills
	}
	if r.Skills == nil {
		r.Skills = hero_skills.NewHeroSkills(r.ID)
		if r.src.Skills != nil {
			r.copyLock.RLock()
			r.Skills.Copy(r.src.Skills)
			r.copyLock.RUnlock()
		}
		r.Skills.Init()
	}
	return r.Skills
}

// GetSkillCollections 获取技能收藏模块
func (r *Role) GetSkillCollections() *hero_skillcollections.HeroSkillCollections {
	if !r.IsCopy() {
		return r.SkillCollections
	}
	if r.SkillCollections == nil {
		r.SkillCollections = hero_skillcollections.NewHeroSkillCollections(r.ID)
		if r.src.SkillCollections != nil {
			r.copyLock.RLock()
			r.SkillCollections.Copy(r.src.SkillCollections)
			r.copyLock.RUnlock()
		}
		r.SkillCollections.Init()
	}
	return r.SkillCollections
}

// GetCultivateCosts 获取养成消耗模块
func (r *Role) GetCultivateCosts() *cultivate_costs.CultivateCosts {
	if !r.IsCopy() {
		return r.CultivateCosts
	}
	if r.CultivateCosts == nil {
		r.CultivateCosts = cultivate_costs.NewCultivateCosts(r.ID)
		if r.src.CultivateCosts != nil {
			r.copyLock.RLock()
			r.CultivateCosts.Copy(r.src.CultivateCosts)
			r.copyLock.RUnlock()
		}
		r.CultivateCosts.Init()
	}
	return r.CultivateCosts
}

// GetItems 获取背包模块
func (r *Role) GetItems() *role_items.RoleItems {
	if !r.IsCopy() {
		return r.Items
	}
	if r.Items == nil {
		r.Items = role_items.NewRoleItems(r.ID)
		if r.src.Items != nil {
			r.copyLock.RLock()
			r.Items.Copy(r.src.Items)
			r.copyLock.RUnlock()
		}
		r.Items.Init()
	}
	return r.Items
}

// GetBuildings 获取建筑模块（主城/分城/军事建筑统一）
func (r *Role) GetBuildings() *role_buildings.RoleBuildings {
	if !r.IsCopy() {
		return r.Buildings
	}
	if r.Buildings == nil {
		r.Buildings = role_buildings.NewRoleBuildings(r.ID)
		if r.src.Buildings != nil {
			r.copyLock.RLock()
			r.Buildings.Copy(r.src.Buildings)
			r.copyLock.RUnlock()
		}
		r.Buildings.Init()
	}
	return r.Buildings
}

// GetFormations 获取上阵队伍模块
func (r *Role) GetFormations() *role_formations.RoleFormations {
	if !r.IsCopy() {
		return r.Formations
	}
	if r.Formations == nil {
		r.Formations = role_formations.NewRoleFormations(r.ID)
		if r.src.Formations != nil {
			r.copyLock.RLock()
			r.Formations.Copy(r.src.Formations)
			r.copyLock.RUnlock()
		}
		r.Formations.Init()
	}
	return r.Formations
}

// GetRecruits 获取抽卡模块
func (r *Role) GetRecruits() *role_recruits.RoleRecruits {
	if !r.IsCopy() {
		return r.Recruits
	}
	if r.Recruits == nil {
		r.Recruits = role_recruits.NewRoleRecruits(r.ID)
		if r.src.Recruits != nil {
			r.copyLock.RLock()
			r.Recruits.Copy(r.src.Recruits)
			r.copyLock.RUnlock()
		}
		r.Recruits.Init()
	}
	return r.Recruits
}

// GetAttr 获取属性模块
func (r *Role) GetAttr() *role_attrs.RoleAttrs {
	if !r.IsCopy() {
		return r.Attr
	}
	if r.Attr == nil {
		r.Attr = role_attrs.NewRoleAttrs(r.ID)
		if r.src.Attr != nil {
			r.copyLock.RLock()
			r.Attr.Copy(r.src.Attr)
			r.copyLock.RUnlock()
		}
		r.Attr.Init()
	}
	return r.Attr
}
