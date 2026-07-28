package hero_skillcollections

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func (hsc *HeroSkillCollections) Init() {
	for _, one := range hsc.List {
		_ = NewHeroSkillCollection(one)
	}
}

func (hsc *HeroSkillCollections) Copy(src *HeroSkillCollections) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, hsc)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	hsc.Init()
}

func (hsc *HeroSkillCollections) Format2Pb() []*pb_skill.SkillCollection {
	list := make([]*pb_skill.SkillCollection, 0, len(hsc.List))
	for _, v := range hsc.List {
		item := NewHeroSkillCollection(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

//-------------------------------

func NewHeroSkillCollection(one *game_models.HeroSkillCollection) *HeroSkillCollection {
	e := &HeroSkillCollection{
		HeroSkillCollection: one,
	}
	return e
}

func (e *HeroSkillCollection) Format2Pb() *pb_skill.SkillCollection {
	if e.HeroSkillCollection == nil {
		return nil
	}
	return &pb_skill.SkillCollection{
		ConfigId:         e.SkillConfID,
		CollectionLevel:  e.CollectionLevel,
		IsUnlock:         e.IsUnlocked,
	}
}
