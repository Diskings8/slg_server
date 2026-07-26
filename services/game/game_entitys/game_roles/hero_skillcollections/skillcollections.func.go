package hero_skillcollections

import (
	"go.uber.org/zap"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_bytes"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

var collectionPool = util_bytes.NewPool(func() *HeroSkillCollections {
	return NewHeroSkillCollections(0)
})

func Get() *HeroSkillCollections {
	return collectionPool.Get()
}

func Release(hsc *HeroSkillCollections) {
	collectionPool.Put(hsc)
}

func (hsc *HeroSkillCollections) Reset() {
	hsc.RoleID = 0
	hsc.List = nil
}

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

func (hsc *HeroSkillCollections) Format2Pb() any {
	// todo
	return nil
}

//-------------------------------

func NewHeroSkillCollection(one *game_models.HeroSkillCollection) *HeroSkillCollection {
	e := &HeroSkillCollection{
		HeroSkillCollection: one,
	}
	return e
}

func (e *HeroSkillCollection) Format2Pb() any {
	// todo
	return nil
}
