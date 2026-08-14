package role_heroes

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/game_conf"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/loggers"
	"server.slg.com/common/models"
	"server.slg.com/common/utils/snowflakes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewRoleHeroes(roleID uint64) *RoleHeroes {
	return &RoleHeroes{
		RoleID: roleID,
		List:   make([]*game_models.RoleHero, 0),
	}
}

func (hrs *RoleHeroes) Init() {
	for _, modelOne := range hrs.List {
		roleHero := NewRoleHero(modelOne)
		hrs.Mem.Store(roleHero.ID, roleHero)
	}
}

func (hrs *RoleHeroes) Copy(src *RoleHeroes) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, hrs)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	hrs.Init()
}

func (hrs *RoleHeroes) Format2Pb() []*pb_hero.HeroInfo {
	list := make([]*pb_hero.HeroInfo, 0, len(hrs.List))
	for _, v := range hrs.List {
		item := NewRoleHero(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

// GetHero 获取英雄（按英雄实例ID，uint64 无截断）
func (hrs *RoleHeroes) GetHero(heroID uint64) *RoleHero {
	if v, ok := hrs.Mem.Load(heroID); ok {
		return v
	}
	return nil
}

// GetHeroesByConf 获取所有指定配置ID的英雄（同配置可多张，用于重复卡/突破等玩法）
func (hrs *RoleHeroes) GetHeroesByConf(confID int32) []*RoleHero {
	out := make([]*RoleHero, 0)
	for _, v := range hrs.List {
		if v.HeroConfID == confID {
			out = append(out, NewRoleHero(v))
		}
	}
	return out
}

// RemoveHero 从内存中移除指定英雄卡（List + Mem 索引）
//
// 仅移除内存，DB 删除需另行调用 DBDeleteHero（DBSave 是 upsert 不会清理 List 外记录）。
func (hrs *RoleHeroes) RemoveHero(heroID uint64) *RoleHero {
	hrs.Mem.Delete(heroID)
	for i, v := range hrs.List {
		if v.ID == heroID {
			out := hrs.List[i]
			hrs.List = append(hrs.List[:i], hrs.List[i+1:]...)
			return NewRoleHero(out)
		}
	}
	return nil
}

// AddHero 新增英雄卡（抽卡产出）
//
// 生成雪花ID构造新英雄（默认等级1/星级0/无属性点），同时写入 List 与 Mem 索引。
// DB 落库由 Role.DBSave 反射 Save 自动 upsert，无需单独调用。
func (hrs *RoleHeroes) AddHero(heroConfID int32) *RoleHero {
	now := time.Now().Unix()
	modelOne := &game_models.RoleHero{
		ModelBase: models.ModelBase{
			ID:        snowflakes.GenUUID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		RoleID:     hrs.RoleID,
		HeroConfID: heroConfID,
		Level:      1,
		Cultivates: make([]*pb_cultivate.Cultivate, 0),
	}
	one := NewRoleHero(modelOne)
	one.RefreshCurVal() // 初始化 cur_val（lv1 → 基础属性）
	hrs.List = append(hrs.List, modelOne)
	hrs.Mem.Store(modelOne.ID, one) // Mem 键为 uint64（雪花ID），与 Init/GetHero/RemoveHero 一致，勿转 int32
	return one
}

//-------------------------------

func NewRoleHero(one *game_models.RoleHero) *RoleHero {
	return &RoleHero{
		RoleHero: one,
	}
}

// ensureCultivates 补齐 5 维 Cultivate（攻/防/智/移/拆），不足补空项
func ensureCultivates(list []*pb_cultivate.Cultivate, n int) []*pb_cultivate.Cultivate {
	for len(list) < n {
		list = append(list, &pb_cultivate.Cultivate{})
	}
	return list
}

// RefreshCurVal 按当前等级重算 5 维 cur_val（等级派生基础属性 = base + growth×(level-1)）。
//
// 由 game 侧在升级/创建时调用；只写 CurVal，不动 add_val_camp 等加点/来源组件。
// battle 节点只读快照里的这些组件求和，不再读英雄配置表。
func (hr *RoleHero) RefreshCurVal() {
	if hr == nil || hr.RoleHero == nil {
		return
	}
	attr := game_conf.Load().Hero.CalcCurVal(hr.HeroConfID, hr.Level)
	hr.Cultivates = ensureCultivates(hr.Cultivates, 5)
	hr.Cultivates[0].CurVal = attr.Attack
	hr.Cultivates[1].CurVal = attr.Defense
	hr.Cultivates[2].CurVal = attr.Intelligence
	hr.Cultivates[3].CurVal = attr.Movement
	hr.Cultivates[4].CurVal = attr.Relocation
}

// curValReady 是否已具备 5 维 cur_val（battle 快照前需要；遗留数据缺 cur_val 会补算）
func (hr *RoleHero) curValReady() bool {
	if hr == nil || hr.RoleHero == nil {
		return false
	}
	if len(hr.Cultivates) < 5 {
		return false
	}
	return hr.Cultivates[0].GetCurVal() != 0
}

func (hr *RoleHero) Format2Pb() *pb_hero.HeroInfo {
	if hr.RoleHero == nil {
		return nil
	}

	// 兜底：遗留英雄未维护 cur_val → 补算，保证快照到 battle 时属性正确
	if !hr.curValReady() {
		hr.RefreshCurVal()
	}

	// Cultivates 按索引映射到具体属性字段:
	//   [0]=AttrAttack, [1]=AttrDefense, [2]=AttrIntelligence, [3]=AttrMovement, [4]=AttrRelocation
	var attrAttack, attrDefense, attrIntelligence, attrMovement, attrRelocation *pb_cultivate.Cultivate
	if len(hr.Cultivates) > 0 {
		attrAttack = hr.Cultivates[0]
	}
	if len(hr.Cultivates) > 1 {
		attrDefense = hr.Cultivates[1]
	}
	if len(hr.Cultivates) > 2 {
		attrIntelligence = hr.Cultivates[2]
	}
	if len(hr.Cultivates) > 3 {
		attrMovement = hr.Cultivates[3]
	}
	if len(hr.Cultivates) > 4 {
		attrRelocation = hr.Cultivates[4]
	}

	return &pb_hero.HeroInfo{
		HeroId:           hr.ID, // 英雄实例ID（此前遗漏，导致 HeroList 拿不到 hero_id）
		ConfigId:         hr.HeroConfID,
		StarStage:        hr.StarStage,
		CurLevel:         hr.Level,
		CurExp:           hr.Exp,
		AttrPoint:        hr.AttrPoint,
		CurStatus:        pb_hero.Status_Normal,
		AttrAttack:       attrAttack,
		AttrDefense:      attrDefense,
		AttrIntelligence: attrIntelligence,
		AttrMovement:     attrMovement,
		AttrRelocation:   attrRelocation,
		Skills:           hr.EquipSkills,
		IsAwakened:       hr.IsAwakened,
		IsLocked:         hr.IsLocked, // 锁定是独立保护标记，不等同于受伤
		Troops:           hr.Troops,
		CurTroopTypeId:   hr.CurTroopTypeID,
	}
}

// GetEquipSkillBySlot 获取英雄技能槽指定槽位装配的技能（slot_id 匹配；无则 nil）
func (hr *RoleHero) GetEquipSkillBySlot(slot int32) *pb_skill.Skill {
	for _, s := range hr.EquipSkills {
		if s.GetSlotId() == slot {
			return s
		}
	}
	return nil
}
