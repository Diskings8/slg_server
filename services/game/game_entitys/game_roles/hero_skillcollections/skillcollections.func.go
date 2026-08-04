package hero_skillcollections

import (
	"time"

	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_skill"
	"server.slg.com/common/loggers"
	"server.slg.com/common/models"
	"server.slg.com/common/utils/snowflakes"
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

//-------------------------------
// 收藏业务方法

// GetBySkillConfID 按技能配置ID查询收藏（遍历 List，无 Mem 索引）
func (hsc *HeroSkillCollections) GetBySkillConfID(skillConfID int32) *HeroSkillCollection {
	for _, one := range hsc.List {
		if one.SkillConfID == skillConfID {
			return NewHeroSkillCollection(one)
		}
	}
	return nil
}

// AddSkillCollection 新增收藏记录（已存在返回 nil；默认未解锁、无收集进度）
func (hsc *HeroSkillCollections) AddSkillCollection(skillConfID int32) *HeroSkillCollection {
	if hsc.GetBySkillConfID(skillConfID) != nil {
		return nil
	}
	now := time.Now().Unix()
	modelOne := &game_models.HeroSkillCollection{
		ModelBase: models.ModelBase{
			ID:        snowflakes.GenUUID(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		RoleID:      hsc.RoleID,
		SkillConfID: skillConfID,
	}
	one := NewHeroSkillCollection(modelOne)
	hsc.List = append(hsc.List, modelOne)
	return one
}
