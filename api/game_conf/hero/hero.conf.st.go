// Package hero 英雄配置表（Go 内嵌占位数据，后续可迁 JSON）
package hero

// Conf 英雄配置聚合
type Conf struct {
	MaxLevel        uint32 // 英雄等级上限
	FreePointPer10L uint32 // 每10级获得的自由属性点
	MaxStarStage    int32  // 星级上限
}

// New 构造英雄配置（内置占位数据）
func New() *Conf {
	return &Conf{
		MaxLevel:        100,
		FreePointPer10L: 5,
		MaxStarStage:    5,
	}
}

// NeedExp 升级所需经验（占位公式，后续接配置表按等级读取）
func (c *Conf) NeedExp(level uint32) uint32 {
	return level * 100
}
