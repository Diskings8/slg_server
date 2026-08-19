// Package guard 地块守军配置表（开发行军的战斗对象，数据源：tabtoy 单一 gameconfig.json，经 NewFromPB 构建）
//
// 开发行军（MarchTypeDevelop）到达地块后，与目标等级（当前等级+3）的守军战斗。
// 守军按地块等级配置：每个等级一组守军队伍（英雄配置ID + 固定兵力）。
// 仅配置了守军的等级为可开发等级，MaxDevelopLevel 为地块可开发的最高等级上限。
package guard

// GuardSlot 守军槽位：单个守军英雄 + 固定兵力
type GuardSlot struct {
	HeroConfID int32 `json:"hero_conf_id"` // 英雄配置ID
	SoldierNum uint32 `json:"soldier_num"` // 固定兵力
}

// GuardConfig 单等级守军配置（亦为 JSON 表行结构）
type GuardConfig struct {
	Level int32        `json:"level"` // 地块等级（守军查表索引）
	Slots []GuardSlot  `json:"slots"` // 守军队伍（英雄槽位列表）
}

// Conf 守军配置聚合
type Conf struct {
	configs         map[int32]*GuardConfig // 等级 → 守军配置
	MaxDevelopLevel int32                  // 地块可开发的最高等级（含）
}

// GetGuard 按地块等级查询守军配置（未配置返回 nil）
func (c *Conf) GetGuard(level int32) *GuardConfig {
	return c.configs[level]
}

// GetMaxDevelopLevel 地块可开发的最高等级
func (c *Conf) GetMaxDevelopLevel() int32 {
	return c.MaxDevelopLevel
}
