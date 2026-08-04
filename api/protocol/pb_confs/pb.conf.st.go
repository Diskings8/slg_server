package pb_confs

type Table struct {
}

type ItemID int32

type ItemType int32

const (
	// ItemTypeNormal 普通道具（背包物品）
	ItemTypeNormal ItemType = iota
	// ItemTypeCurrency1 一级货币（仅充值获得，如钻石）
	ItemTypeCurrency1
	// ItemTypeCurrency2 二级货币（游戏内主要消耗用途，如金币）
	ItemTypeCurrency2
)

type ItemSubType int32

// ── 货币配置ID占位约定（后续接入配置表） ──
const (
	Currency1ConfID ItemID = 100001 // 一级货币配置ID
	Currency2ConfID ItemID = 100002 // 二级货币配置ID
)
