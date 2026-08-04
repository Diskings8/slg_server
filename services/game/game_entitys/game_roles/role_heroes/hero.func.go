package role_heroes

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb/pb_hero"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
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
		hrs.Mem.Store(pb_confs.ItemID(roleHero.ID), roleHero)
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

// GetHero 获取英雄（按英雄实例ID）
func (hrs *RoleHeroes) GetHero(heroID pb_confs.ItemID) *RoleHero {
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

//-------------------------------

func NewRoleHero(one *game_models.RoleHero) *RoleHero {
	return &RoleHero{
		RoleHero: one,
	}
}

func (hr *RoleHero) Format2Pb() *pb_hero.HeroInfo {
	if hr.RoleHero == nil {
		return nil
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

	// IsLocked 反映到 cur_status
	status := pb_hero.Status_Normal
	if hr.IsLocked {
		status = pb_hero.Status_Injured
	}

	return &pb_hero.HeroInfo{
		ConfigId:         hr.HeroConfID,
		CurLevel:         hr.Level,
		CurExp:           hr.Exp,
		AttrPoint:        hr.AttrPoint,
		CurStatus:        status,
		AttrAttack:       attrAttack,
		AttrDefense:      attrDefense,
		AttrIntelligence: attrIntelligence,
		AttrMovement:     attrMovement,
		AttrRelocation:   attrRelocation,
		Skills:           hr.EquipSkills,
		IsAwakened:       hr.IsAwakened,
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
