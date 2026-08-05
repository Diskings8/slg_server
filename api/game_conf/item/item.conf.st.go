// Package item 道具配置表（Go 内嵌占位数据，后续可迁 JSON）
package item

import "server.slg.com/api/protocol/pb_confs"

// ItemEffectType 道具效果类型
type ItemEffectType int32

const (
	// EffectNone 无效果（仅消耗）
	EffectNone ItemEffectType = iota
	// EffectAddHeroExp 加英雄经验（Target 忽略，目标英雄由使用请求指定；Value=单个道具经验）
	EffectAddHeroExp
	// EffectAddCurrency 加货币（Target=货币配置ID；Value=单个道具数量）
	EffectAddCurrency
	// EffectAddItem 资源包，加道具（Target=道具配置ID；Value=单个道具数量）
	EffectAddItem
)

// ItemEffect 道具效果定义
type ItemEffect struct {
	Type   ItemEffectType `json:"type"`
	Target int32          `json:"target"` // 目标配置ID（货币/道具）
	Value  int64          `json:"value"`  // 单个道具的效果数值
}

// ItemConfig 道具配置
type ItemConfig struct {
	ConfID pb_confs.ItemID `json:"conf_id"`
	Effect ItemEffect      `json:"effect"`
}

// Conf 道具配置聚合
type Conf struct {
	configs map[pb_confs.ItemID]ItemConfig

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造道具配置（内置占位数据）
func New() *Conf {
	return &Conf{
		configs: map[pb_confs.ItemID]ItemConfig{
			2001: {ConfID: 2001, Effect: ItemEffect{Type: EffectAddHeroExp, Value: 100}},                                              // 英雄经验书
			2002: {ConfID: 2002, Effect: ItemEffect{Type: EffectAddCurrency, Target: int32(pb_confs.Currency2ConfID), Value: 1000}}, // 金币礼包
			2003: {ConfID: 2003, Effect: ItemEffect{Type: EffectAddItem, Target: 2001, Value: 5}},                                   // 资源包：5 张经验书
			2004: {ConfID: 2004, Effect: ItemEffect{Type: EffectNone}},                                                              // 抽卡券（无效果，仅消耗）
		},
	}
}

// Get 按配置ID查询道具配置
func (c *Conf) Get(configID pb_confs.ItemID) (ItemConfig, bool) {
	ic, ok := c.configs[configID]
	return ic, ok
}

// Has 判断道具配置是否存在（供跨表引用校验）。
func (c *Conf) Has(configID pb_confs.ItemID) bool {
	_, ok := c.configs[configID]
	return ok
}
