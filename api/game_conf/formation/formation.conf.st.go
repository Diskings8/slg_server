// Package formation 编队配置表（Go 内嵌占位数据，后续可迁 JSON）
package formation

// Conf 编队配置聚合
type Conf struct {
	MaxSlots int // 编队英雄槽位上限（0=大营，1/2=前后排；默认 3）

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造编队配置（内置占位数据）
func New() *Conf {
	return &Conf{MaxSlots: 3}
}
