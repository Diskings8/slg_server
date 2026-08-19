// Package formation 编队配置表（数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
package formation

// Conf 编队配置聚合
type Conf struct {
	MaxSlots int // 编队英雄槽位上限（0=大营，1/2=前后排；默认 3）
}
