package common_declarations

import (
	"server.slg.com/api/protocol/pb/pb_item"
	"server.slg.com/api/protocol/pb_confs"
)

type NodeService int

const (
	NodeGameService     NodeService = 10
	NodeGatewayService  NodeService = 20
	NodeWorldMapService NodeService = 30
	NodeBattleService   NodeService = 40
)

type LoaderFunc[M DataI] func(id uint64) (M, error)

// ItemChangeReason 道具变动原因
type ItemChangeReason string

func (icr ItemChangeReason) ToString() string {
	return string(icr)
}

const (
	ReasonUse      ItemChangeReason = "use"      // 使用/消耗
	ReasonReward   ItemChangeReason = "reward"   // 奖励/获得
	ReasonPurchase ItemChangeReason = "purchase" // 购买
	ReasonDrop     ItemChangeReason = "drop"     // 丢弃
	ReasonAdmin    ItemChangeReason = "admin"    // 后台发放/扣除
	ReasonSell     ItemChangeReason = "sell"     // 出售
	ReasonMerge    ItemChangeReason = "merge"    // 合成/融合
	ReasonSplit    ItemChangeReason = "split"    // 拆分
)

type ItemUse struct {
	ItemID      pb_confs.ItemID
	ItemType    pb_confs.ItemType
	ItemSubType pb_confs.ItemSubType
	Count       int64
}

func (item ItemUse) Format2Pb() *pb_item.ItemUse {
	return &pb_item.ItemUse{
		ConfId: int32(item.ItemID),
		Count:  item.Count,
	}
}

func (item ItemUse) Format2ChangeLogPb(optID string, roleID uint64, modifyCount, curCount, optTimeUx int64, reason string) *pb_item.ItemChangeLog {
	return &pb_item.ItemChangeLog{
		OptId:       optID,
		ConfigId:    int32(item.ItemID),
		ItemType:    int32(item.ItemType),
		ItemSubType: int32(item.ItemSubType),
		Delta:       modifyCount,
		Balance:     curCount,
		Reason:      reason,
		RoleId:      roleID,
		Timestamp:   optTimeUx,
	}
}
