// Package item 道具配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
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
