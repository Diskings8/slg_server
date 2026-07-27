package hero_skills

import (
	"go.uber.org/zap"
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

func (hss *HeroSkills) Format2Pb() any {
	// todo
	return nil
}

//-------------------------------

func NewHeroSkill(one *game_models.HeroSkill) *HeroSkill {
	h := &HeroSkill{
		HeroSkill: one,
	}
	// to add other attr
	return h
}

func (hs *HeroSkill) Format2Pb() any {
	// todo
	return nil
}
