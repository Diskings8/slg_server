package hero_skills

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewHeroSkills(roleID uint64) *HeroSkills {
	return &HeroSkills{
		RoleID: roleID,
		List:   make([]*game_models.HeroSkill, 0),
	}
}

func (hss *HeroSkills) Init() {
	for _, modelOne := range hss.List {
		heroSkill := NewHeroSkill(modelOne)
		hss.Mem.Store(pb_confs.ItemID(heroSkill.ID), heroSkill)
	}
}

func (hss *HeroSkills) Copy(src *HeroSkills) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, hss)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	hss.Init()
}

func (hss *HeroSkills) Format2Pb() []*pb_skill.Skill {
	list := make([]*pb_skill.Skill, 0, len(hss.List))
	for _, v := range hss.List {
		item := NewHeroSkill(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

//-------------------------------

func NewHeroSkill(one *game_models.HeroSkill) *HeroSkill {
	h := &HeroSkill{
		HeroSkill: one,
	}
	// to add other attr
	return h
}

func (hs *HeroSkill) Format2Pb() *pb_skill.Skill {
	if hs.HeroSkill == nil {
		return nil
	}
	return &pb_skill.Skill{
		ConfigId:       hs.SkillConfID,
		Level:          hs.Level,
		Research_Level: hs.ResearchLevel,
		IsUnlock:       hs.IsUnlocked,
	}
}
