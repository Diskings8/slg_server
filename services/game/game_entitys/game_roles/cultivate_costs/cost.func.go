package cultivate_costs

import (
	"go.uber.org/zap"
	"server.slg.com/api/protocol/pb/pb_cultivate"
	"server.slg.com/api/protocol/pb_confs"
	"server.slg.com/common/loggers"
	"server.slg.com/common/utils/util_jsons"
	"server.slg.com/services/game/game_models"
)

func NewCultivateCosts(roleID uint64) *CultivateCosts {
	return &CultivateCosts{
		RoleID: roleID,
		List:   make([]*game_models.CultivateCost, 0),
	}
}

func (ccs *CultivateCosts) Init() {
	for _, modelOne := range ccs.List {
		cultivateCost := NewCultivateCost(modelOne)
		ccs.Mem.Store(pb_confs.ItemID(cultivateCost.ID), cultivateCost)
	}
}

func (ccs *CultivateCosts) Copy(src *CultivateCosts) {
	b, err := util_jsons.Marshal(src)
	if err != nil {
		loggers.Logger.Error("marshal failed", zap.Any("src", src), zap.Error(err))
	}

	err = util_jsons.Unmarshal(b, ccs)
	if err != nil {
		loggers.Logger.Error("unmarshal failed", zap.Any("src", src), zap.Error(err))
	}

	ccs.Init()
}

func (ccs *CultivateCosts) Format2Pb() []*pb_cultivate.CultivateCost {
	list := make([]*pb_cultivate.CultivateCost, 0, len(ccs.List))
	for _, v := range ccs.List {
		item := NewCultivateCost(v)
		list = append(list, item.Format2Pb())
	}
	return list
}

//-------------------------------

func NewCultivateCost(one *game_models.CultivateCost) *CultivateCost {
	return &CultivateCost{
		CultivateCost: one,
	}
}

func (cc *CultivateCost) Format2Pb() *pb_cultivate.CultivateCost {
	if cc.CultivateCost == nil {
		return nil
	}

	pb := &pb_cultivate.CultivateCost{}

	// 根据养成类型将 Cost 分流到对应的字段
	switch cc.CultivateType {
	case pb_cultivate.CultivateType_CultivateSkill:
		pb.SkillUse = cc.Cost
	case pb_cultivate.CultivateType_CultivateTroop:
		pb.TroopUse = cc.Cost
	}

	return pb
}
