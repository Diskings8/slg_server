// Package resource 资源地产量配置表（每小时产量，扫荡采集/占领收益基础）
//
// 产出模型：
//   - lv1：混合型 — 全 4 项资源各产 amount（每小时）
//   - lv2：双资源 — 主资源 + 随机一项次级资源，各产对应 amount
//   - lv3~9：单项 — 只产主资源类型，产量逐级增加（lv5 为关键分水岭，= lv4×3）
//
// 开发行军对 lv2~lv4 资源地统一 +3 等级（developStep=3），不在此表配目标等级。
package resource

// ResourceType 资源地产出方式
type ResourceType int32

const (
	// ResourceTypeMixed 混合型：全类型各产 Amount（lv1）
	ResourceTypeMixed ResourceType = 0
	// ResourceTypeDual 双资源：主资源 + 随机次级资源（lv2）
	ResourceTypeDual ResourceType = 1
	// ResourceTypeSingle 单项：只产主资源（lv3+）
	ResourceTypeSingle ResourceType = 2
)

// ResourceConfig 单等级资源产量配置（亦为 JSON 表行结构）
type ResourceConfig struct {
	Level int32 `json:"level"` // 资源地等级 1~9
	Type  int32 `json:"type"`  // ResourceType

	// mixed/single：Amount 为产量（mixed=每类型各产；single=主资源产量）
	// dual：PrimaryAmount/SecondaryAmount 为主/次级资源各自产量
	Amount          int32 `json:"amount"`
	PrimaryAmount   int32 `json:"primary_amount"`
	SecondaryAmount int32 `json:"secondary_amount"`
}

// Conf 资源产量配置聚合
type Conf struct {
	configs map[int32]*ResourceConfig // 等级 → 产量配置

	version string // 内容版本（JSON 加载后为内容 hash；内嵌为 ""）
}

// New 构造资源产量配置（内置占位数据）
func New() *Conf {
	return &Conf{
		configs: map[int32]*ResourceConfig{
			1: {Level: 1, Type: int32(ResourceTypeMixed), Amount: 36}, // 全 4 项各 +36/h
			2: {Level: 2, Type: int32(ResourceTypeDual), PrimaryAmount: 120, SecondaryAmount: 120},
			3: {Level: 3, Type: int32(ResourceTypeSingle), Amount: 360},
			4: {Level: 4, Type: int32(ResourceTypeSingle), Amount: 480},
			5: {Level: 5, Type: int32(ResourceTypeSingle), Amount: 1200}, // 关键分水岭
			6: {Level: 6, Type: int32(ResourceTypeSingle), Amount: 1400},
			7: {Level: 7, Type: int32(ResourceTypeSingle), Amount: 1600},
			8: {Level: 8, Type: int32(ResourceTypeSingle), Amount: 1800},
			9: {Level: 9, Type: int32(ResourceTypeSingle), Amount: 2000},
		},
	}
}

// GetConfig 按等级查询配置（未配置返回 nil）
func (c *Conf) GetConfig(level int32) *ResourceConfig {
	return c.configs[level]
}

// GetProduction 主资源产量（mixed=单类型产量；dual=主资源产量；single=主资源产量）
func (c *Conf) GetProduction(level int32) int32 {
	cfg := c.configs[level]
	if cfg == nil {
		return 0
	}
	switch ResourceType(cfg.Type) {
	case ResourceTypeDual:
		return cfg.PrimaryAmount
	default:
		return cfg.Amount
	}
}

// IsMixed 是否为全类型混合产出（lv1）
func (c *Conf) IsMixed(level int32) bool {
	cfg := c.configs[level]
	return cfg != nil && ResourceType(cfg.Type) == ResourceTypeMixed
}

// IsDual 是否为双资源产出（lv2）
func (c *Conf) IsDual(level int32) bool {
	cfg := c.configs[level]
	return cfg != nil && ResourceType(cfg.Type) == ResourceTypeDual
}
